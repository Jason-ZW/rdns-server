package rds

import (
	"database/sql"
	"strings"

	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *RDS) setEmpty(tx *sql.Tx, payload types.Payload, createTime, tokenID int64) (int64, error) {
	name := "empty." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
	result, err := tx.Exec("INSERT INTO record_a (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)",
		name, utils.TypeToInt(payload.Type, false), "", createTime, tokenID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) setRoot(tx *sql.Tx, payload types.Payload, createTime, tokenID int64) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
	if payload.Wildcard {
		name = utils.WildcardEscape(utils.TrimTrailingDot(payload.Fqdn))
	}

	var values string
	switch payload.Type {
	case types.RecordTypeAAAA, types.RecordTypeA:
		values = strings.Join(payload.Hosts, ",")
	case types.RecordTypeTXT:
		values = utils.TextRemoveQuotes(payload.Text)
	default:
		values = payload.Hosts[0]
	}

	result, err := tx.Exec("INSERT INTO record_a (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)",
		name, utils.TypeToInt(payload.Type, false), values, createTime, tokenID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) setWildcard(tx *sql.Tx, payload types.Payload, createTime, tokenID int64) (int64, error) {
	name := "\\052." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	var values string
	switch payload.Type {
	case types.RecordTypeAAAA, types.RecordTypeA:
		values = strings.Join(payload.Hosts, ",")
	case types.RecordTypeTXT:
		values = utils.TextRemoveQuotes(payload.Text)
	default:
		values = payload.Hosts[0]
	}

	result, err := tx.Exec("INSERT INTO record_a (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)",
		name, utils.TypeToInt(payload.Type, false), values, createTime, tokenID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) setSubDomain(tx *sql.Tx, dnsName, dnsType string, value []string, createTime, rootID int64) (int64, error) {
	result, err := tx.Exec("INSERT INTO sub_record_a (fqdn, type, content, created_on, pid) VALUES (?, ?, ?, ?, ?)",
		dnsName, utils.TypeToInt(dnsType, true), strings.Join(value, ","), createTime, rootID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) setTXT(tx *sql.Tx, payload types.Payload, createTime, tokenID int64) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	result, err := tx.Exec("INSERT INTO record_txt (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)",
		name, utils.TypeToInt(payload.Type, false), utils.TextWithQuotes(payload.Text), createTime, tokenID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) setCNAME(tx *sql.Tx, payload types.Payload, createTime, tokenID int64) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	result, err := tx.Exec("INSERT INTO record_cname (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)",
		name, utils.TypeToInt(payload.Type, false), payload.CNAME, createTime, tokenID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) putRoot(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
	if payload.Wildcard {
		name = utils.WildcardEscape(utils.TrimTrailingDot(payload.Fqdn))
	}

	var values string
	switch payload.Type {
	case types.RecordTypeAAAA, types.RecordTypeA:
		values = strings.Join(payload.Hosts, ",")
	case types.RecordTypeTXT:
		values = utils.TextRemoveQuotes(payload.Text)
	default:
		values = payload.Hosts[0]
	}

	result, err := tx.Exec("UPDATE record_a SET content = ? WHERE fqdn = ?", values, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) putWildcard(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := "\\052." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	if payload.Wildcard {
		name = utils.WildcardEscape(utils.TrimTrailingDot(payload.Fqdn))
	}

	var values string
	switch payload.Type {
	case types.RecordTypeAAAA, types.RecordTypeA:
		values = strings.Join(payload.Hosts, ",")
	case types.RecordTypeTXT:
		values = utils.TextRemoveQuotes(payload.Text)
	default:
		values = payload.Hosts[0]
	}

	result, err := tx.Exec("UPDATE record_a SET content = ? WHERE fqdn = ?", values, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) putTXT(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	result, err := tx.Exec("UPDATE record_txt SET content = ? WHERE fqdn = ?", utils.TextWithQuotes(payload.Text), name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) putCNAME(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	result, err := tx.Exec("UPDATE record_cname SET content = ? WHERE fqdn = ?", payload.CNAME, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) putSubDomain(tx *sql.Tx, domain string, content string) (int64, error) {
	result, err := tx.Exec("UPDATE sub_record_a SET content = ? WHERE fqdn = ?", content, domain)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delToken(tx *sql.Tx, payload types.Payload) (int64, error) {
	result, err := tx.Exec("DELETE FROM token WHERE fqdn = ?", payload.Fqdn)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delEmpty(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := "empty." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
	result, err := tx.Exec("DELETE FROM record_a WHERE fqdn = ?", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delRoot(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)
	if payload.Wildcard {
		name = utils.WildcardEscape(utils.TrimTrailingDot(payload.Fqdn))
	}

	result, err := tx.Exec("DELETE FROM record_a WHERE fqdn = ?", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delWildcard(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := "\\052." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	if payload.Wildcard {
		name = utils.WildcardEscape(utils.TrimTrailingDot(payload.Fqdn))
	}

	result, err := tx.Exec("DELETE FROM record_a WHERE fqdn = ?", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delTXT(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	result, err := tx.Exec("DELETE FROM record_txt WHERE fqdn = ?", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delCNAME(tx *sql.Tx, payload types.Payload) (int64, error) {
	name := utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	result, err := tx.Exec("DELETE FROM record_cname WHERE fqdn = ?", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) delSubDomain(tx *sql.Tx, domain string) (int64, error) {
	result, err := tx.Exec("DELETE FROM sub_record_a WHERE fqdn = ?", domain)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *RDS) getEmpty(payload types.Payload) (int64, error) {
	name := "empty." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

	if payload.Wildcard {
		name = "empty." + utils.WildcardEscape(utils.TrimTrailingDot(payload.Fqdn))
	}

	statement, err := r.db.Prepare("SELECT id FROM record_a WHERE fqdn = ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return 0, err
	}

	var result int64
	if err := statement.QueryRow(name).Scan(&result); err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	if result <= 0 {
		return 0, errors.New("failed to found record")
	}

	return result, nil
}

func (r *RDS) getSubDomains(parentID int64) ([]types.Domain, error) {
	output := make([]types.Domain, 0)

	statement, err := r.db.Prepare("SELECT fqdn, type, content FROM sub_record_a WHERE pid = ?")

	defer func() {
		if err := statement.Close(); err != nil {
			logrus.Errorf("close database statement failed: %s\n", err.Error())
		}
	}()

	if err != nil {
		return output, err
	}

	rows, err := statement.Query(parentID)
	if err != nil {
		return output, err
	}

	for rows.Next() {
		var content string
		temp := &types.Domain{}
		if err := rows.Scan(&temp.Fqdn, &temp.Type, &content); err != nil {
			return output, err
		}

		temp.Hosts = strings.Split(content, ",")
		output = append(output, *temp)
	}

	return output, nil
}
