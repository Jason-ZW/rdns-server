package rds

import (
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/pkg/consts"
	"github.com/rancher/rdns-server/pkg/tools/sqlmigrate"
	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	// in order to make build through
	_ "github.com/go-sql-driver/mysql"
)

const (
	keeperName  = "rds"
	tokenLength = 32
)

type RDS struct {
	db     *sql.DB
	domain string
	name   string
	ttl    int64
	expire int64
	rotate int64
}

func NewRDS(domain string, ttl int64) *RDS {
	expire, err := time.ParseDuration(os.Getenv("EXPIRE"))
	if err != nil {
		logrus.WithError(err).Errorln("failed to parse expire")
		return &RDS{}
	}

	rotate, err := time.ParseDuration(os.Getenv("ROTATE"))
	if err != nil {
		logrus.WithError(err).Errorln("failed to parse rotate")
		return &RDS{}
	}

	db, err := sql.Open(consts.DBDriverName, os.Getenv("DB_DSN"))
	if err != nil {
		logrus.Debugf("failed to open database: %s\n", err.Error())
		return nil
	}

	db.SetMaxOpenConns(consts.DBMaxOpenConnections)
	db.SetMaxIdleConns(consts.DBMaxIdleConnections)

	if err := db.Ping(); err != nil {
		logrus.Debugf("failed to connect database: %s\n", err.Error())
		return nil
	}

	switch os.Getenv("DB_MIGRATE") {
	case "up":
		if _, err := sqlmigrate.NewSQLMigrate(db).Up(); err != nil {
			logrus.Errorf("database migrate up failed: %s\n", err.Error())
			return nil
		}
	case "down":
		if _, err := sqlmigrate.NewSQLMigrate(db).Down(); err != nil {
			logrus.Errorf("database migrate down failed: %s\n", err.Error())
			return nil
		}
	}

	return &RDS{
		db:     db,
		domain: domain,
		name:   keeperName,
		ttl:    ttl,
		expire: int64(expire.Seconds()),
		rotate: int64(rotate.Seconds()),
	}
}

func (r *RDS) Close() error {
	return r.db.Close()
}

// PrefixCanBeUsed used to check whether the dns prefix can be used.
func (r *RDS) PrefixCanBeUsed(prefix string) (bool, error) {
	statement, err := r.db.Prepare("SELECT prefix FROM frozen_prefix WHERE prefix = ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return false, err
	}

	var result string
	if err := statement.QueryRow(prefix).Scan(&result); err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if result != "" {
		return false, nil
	}

	return true, nil
}

// IsSubDomain used to check whether the record is sub domain or not.
func (r *RDS) IsSubDomain(dnsName string) bool {
	statement, err := r.db.Prepare("SELECT fqdn FROM sub_record_a WHERE fqdn = ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return false
	}

	var result string
	if err := statement.QueryRow(dnsName).Scan(&result); err != nil && err != sql.ErrNoRows {
		return false
	}

	if result == "" {
		return false
	}

	return true
}

// SetKeeps set key information which maintain records' token and sub domain relationships.
func (r *RDS) SetKeeps(payload types.Payload) (keepers.Keep, error) {
	token := utils.RandStringWithAll(tokenLength)
	prefix := utils.GetDNSPrefix(payload.Fqdn, payload.Wildcard)
	createTime := time.Now()
	createTimeUnix := createTime.UnixNano()

	err := r.Transaction(func(tx *sql.Tx) error {
		frozenQueryResult := tx.QueryRow("SELECT id FROM frozen_prefix WHERE prefix = ?", prefix)
		var frozenID int64
		frozenQueryResult.Scan(&frozenID)
		if frozenID <= 0 {
			if _, err := tx.Exec("INSERT INTO frozen_prefix (prefix, created_on) VALUES ( ?, ? )",
				prefix, createTimeUnix); err != nil {
				return err
			}
		}

		tokenQueryResult := tx.QueryRow("SELECT id FROM token WHERE fqdn = ?", payload.Fqdn)
		var tokenID int64
		tokenQueryResult.Scan(&tokenID)
		if tokenID <= 0 {
			tokenResult, err := tx.Exec("INSERT INTO token (token, fqdn, created_on) VALUES( ?, ?, ? )",
				token, payload.Fqdn, createTimeUnix)
			if err != nil {
				return err
			}

			tokenID, err = tokenResult.LastInsertId()
			if err != nil {
				return err
			}
		}

		switch payload.Type {
		case types.RecordTypeAAAA, types.RecordTypeA:
			// set empty record.
			rootID, err := r.setEmpty(tx, payload, createTimeUnix, tokenID)
			if err != nil {
				return err
			}

			// set root domain record.
			if _, err := r.setRoot(tx, payload, createTimeUnix, tokenID); err != nil {
				return err
			}

			// set wildcard domain record.
			if !payload.Wildcard {
				if _, err := r.setWildcard(tx, payload, createTimeUnix, tokenID); err != nil {
					return err
				}
			}

			// set sub domain record.
			if !payload.Wildcard {
				for k, v := range payload.SubDomain {
					subName := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
					_, err := r.setSubDomain(tx, subName, payload.Type, v, createTimeUnix, rootID)
					if err != nil {
						return err
					}
				}
			}
		case types.RecordTypeTXT:
			// set txt record.
			if _, err := r.setTXT(tx, payload, createTimeUnix, tokenID); err != nil {
				return err
			}
		case types.RecordTypeCNAME:
			// set cname record.
			if _, err := r.setCNAME(tx, payload, createTimeUnix, tokenID); err != nil {
				return err
			}
		default:
			return errors.New("unknown record type")
		}

		return nil
	})

	if err != nil {
		return keepers.Keep{}, err
	}

	return keepers.Keep{
		Domain: payload.Fqdn,
		Type:   payload.Type,
		Token:  token,
		Expire: utils.ConvertExpire(createTime, r.expire),
	}, nil
}

// PutKeeps update key information which maintain records' token and sub domain relationships.
func (r *RDS) PutKeeps(payload types.Payload) (keepers.Keep, error) {
	keep, err := r.GetKeep(payload)
	if err != nil {
		return keepers.Keep{}, err
	}

	err = r.Transaction(func(tx *sql.Tx) error {
		switch payload.Type {
		case types.RecordTypeAAAA, types.RecordTypeA:
			// update root domain record.
			if _, err := r.putRoot(tx, payload); err != nil {
				return err
			}

			// update wildcard domain record.
			if _, err := r.putWildcard(tx, payload); err != nil {
				return err
			}

			// update sub domain record.
			parentID, err := r.getEmpty(payload)
			if err != nil {
				return err
			}

			subDomains, err := r.getSubDomains(parentID)
			if err != nil {
				return err
			}

			for _, s := range subDomains {
				prefix := utils.GetDNSPrefix(s.Fqdn, payload.Wildcard)
				if _, ok := payload.SubDomain[prefix]; !ok {
					// delete sub domain.
					if _, err := r.delSubDomain(tx, s.Fqdn); err != nil {
						return err
					}
				}
			}

			for k, v := range payload.SubDomain {
				isAdd := true

				for _, s := range subDomains {
					prefix := utils.GetDNSPrefix(s.Fqdn, payload.Wildcard)
					if prefix == k {
						isAdd = false
						// update sub domain.
						if _, err := r.putSubDomain(tx, s.Fqdn, strings.Join(s.Hosts, ",")); err != nil {
							return err
						}
						continue
					}
				}

				if isAdd {
					// add sub domain.
					var subName string
					if payload.Wildcard {
						subName = k + "." + utils.GetDNSName(payload.Fqdn)
					} else {
						subName = k + "." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
					}

					if _, err := r.setSubDomain(tx, subName, payload.Type, v, keep.Expire.UnixNano(), parentID); err != nil {
						return err
					}
				}
			}
		case types.RecordTypeTXT:
			// update txt record.
			if _, err := r.putTXT(tx, payload); err != nil {
				return err
			}
		case types.RecordTypeCNAME:
			// update cname record.
			if _, err := r.putCNAME(tx, payload); err != nil {
				return err
			}
		default:
			return errors.New("unknown record type")
		}

		return nil
	})

	if err != nil {
		return keepers.Keep{}, err
	}

	return keep, nil
}

// DeleteKeeps delete key information which maintain records' token and sub domain relationships.
func (r *RDS) DeleteKeeps(payload types.Payload) (keepers.Keep, error) {
	keep, err := r.GetKeep(payload)
	if err != nil {
		return keepers.Keep{}, err
	}

	err = r.Transaction(func(tx *sql.Tx) error {
		switch payload.Type {
		case types.RecordTypeAAAA, types.RecordTypeA:
			// delete empty record.
			if _, err := r.delEmpty(tx, payload); err != nil {
				return err
			}
			// delete root domain record.
			if _, err := r.delRoot(tx, payload); err != nil {
				return err
			}
			// delete wildcard domain record.
			if _, err := r.delWildcard(tx, payload); err != nil {
				return err
			}
		case types.RecordTypeTXT:
			// delete txt record.
			if _, err := r.delTXT(tx, payload); err != nil {
				return err
			}
		case types.RecordTypeCNAME:
			// delete cname record.
			if _, err := r.delCNAME(tx, payload); err != nil {
				return err
			}
		default:
			return errors.New("unknown record type")
		}

		return nil
	})

	if err != nil {
		return keepers.Keep{}, err
	}

	return keep, nil
}

// RenewKeeps renew key information which maintain records' token and sub domain relationships.
func (r *RDS) RenewKeeps(payload types.Payload) error {
	createTime := time.Now()
	createTimeUnix := createTime.UnixNano()

	err := r.Transaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec("UPDATE token SET created_on = ? WHERE fqdn = ?",
			createTimeUnix, utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)); err != nil {
			return err
		}

		if _, err := tx.Exec("UPDATE frozen_prefix SET created_on = ? WHERE prefix = ?",
			createTimeUnix, utils.GetDNSPrefix(payload.Fqdn, payload.Wildcard)); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// GetTokenCount returns token count
func (r *RDS) GetTokenCount() int64 {
	var count int64

	statement, err := r.db.Prepare("SELECT count(*) FROM token")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return count
	}

	if err := statement.QueryRow().Scan(&count); err != nil {
		return count
	}

	return count
}

func (r *RDS) DeleteExpiredRotate(t *time.Time) error {
	statement, err := r.db.Prepare("DELETE FROM frozen_prefix WHERE created_on <= ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return err
	}

	_, err = statement.Exec(t.UnixNano())

	return err
}

// GetExpiredTokens return tokens which are expired.
func (r *RDS) GetExpiredTokens(t *time.Time) ([]keepers.Keep, error) {
	keeps := make([]keepers.Keep, 0)

	statement, err := r.db.Prepare("SELECT * FROM token WHERE created_on <= ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return keeps, err
	}

	rows, err := statement.Query(t.UnixNano())
	if err != nil {
		return keeps, err
	}

	for rows.Next() {
		keep := keepers.Keep{}
		if err := rows.Scan(&keep.ID, &keep.Token, &keep.Domain, &keep.Expire); err != nil {
			return keeps, err
		}
		keeps = append(keeps, keep)
	}

	return keeps, nil
}

// DeleteExpiredTokens delete token which is expired.
func (r *RDS) DeleteExpiredTokens(t *time.Time) error {
	statement, err := r.db.Prepare("DELETE FROM token WHERE created_on <= ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return err
	}

	_, err = statement.Exec(t.UnixNano())

	return err
}

// GetKeep get key information which maintain records' token and sub domain relationships.
func (r *RDS) GetKeep(payload types.Payload) (keepers.Keep, error) {
	var dnsName string

	if payload.Wildcard {
		dnsName = utils.GetDNSName(payload.Fqdn)
	} else {
		dnsName = utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
	}

	statement, err := r.db.Prepare("SELECT * FROM token WHERE fqdn = ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return keepers.Keep{}, err
	}

	var createTime int64
	result := &keepers.Keep{}
	if err := statement.QueryRow(dnsName).Scan(&result.ID, &result.Token, &result.Domain, &createTime); err != nil && err != sql.ErrNoRows {
		return keepers.Keep{}, err
	}

	t := time.Unix(0, createTime)
	result.Expire = &t
	result.Expire = utils.ConvertExpire(*result.Expire, r.expire)

	return *result, nil
}

// Transaction used to encapsulating database transactions.
func (r *RDS) Transaction(txFunc func(*sql.Tx) error) (err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				logrus.Errorf("database transaction rollback failed: %s\n", err.Error())
			}
			panic(p)
		} else if err != nil {
			logrus.Errorf("database action failed: %+s\n", err.Error())
			if err := tx.Rollback(); err != nil {
				logrus.Errorf("database transaction rollback failed: %s\n", err.Error())
			}
		} else {
			if err := tx.Commit(); err != nil {
				logrus.Errorf("database transaction commit failed: %s\n", err.Error())
			}
		}
	}()

	err = txFunc(tx)
	return err
}
