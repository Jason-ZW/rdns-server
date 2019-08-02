package rds

import (
	"database/sql"
	"os"
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/pkg/consts"
	"github.com/rancher/rdns-server/pkg/tools/sqlmigrate"
	"github.com/rancher/rdns-server/types"

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
	statement, err := r.db.Prepare("SELECT prefix FROM rdns_prefix where prefix = ?")
	if err != nil {
		return false, err
	}

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err)
		}
	}()

	var result string
	if err := statement.QueryRow(prefix).Scan(&result); err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if result == "" {
		return false, nil
	}

	return true, nil
}

func (r *RDS) SetTransaction(payload types.Payload) (keepers.Keep, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return keepers.Keep{}, err
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			logrus.Error("rollback database transaction failed: %s\n", err)
		}
	}()

	tx.Prepare("INSERT INTO rdns_token (token, domain, type, created_on) VALUES( ?, ?, ? )")


	if err := tx.Commit(); err != nil {
		return keepers.Keep{}, err
	}

	return keepers.Keep{
		Domain: payload.Domain,
		Type:   payload.Type,
		Token:  "",
		Expire: "",
	}, err
}
