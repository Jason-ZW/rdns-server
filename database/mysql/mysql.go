package mysql

import (
	"database/sql"
	"time"

	"github.com/rancher/rdns-server/model"

	// in order to make build through
	_ "github.com/go-sql-driver/mysql"
)

const (
	DataBaseName = "mysql"

	maxOpenConnections = 2000
	maxIdleConnections = 1000
)

type Database struct {
	Db *sql.DB
}

func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open(DataBaseName, dsn)
	if err != nil {
		return &Database{}, err
	}

	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxIdleConnections)

	if err := db.Ping(); err != nil {
		return &Database{}, err
	}

	return &Database{db}, err
}

func (d *Database) InsertFrozen(prefix string) error {
	st, err := d.Db.Prepare("INSERT INTO frozen_prefix (prefix, created_on) VALUES ( ?, ? )")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(prefix, time.Now().Unix())
	return err
}

func (d *Database) QueryFrozen(prefix string) (string, error) {
	st, err := d.Db.Prepare("SELECT prefix FROM frozen_prefix WHERE prefix = ?")
	if err != nil {
		return "", err
	}
	defer st.Close()

	var result string
	if err := st.QueryRow(prefix).Scan(&result); err != nil {
		return "", err
	}

	return result, nil
}

func (d *Database) RenewFrozen(prefix string) error {
	st, err := d.Db.Prepare("UPDATE frozen_prefix SET created_on = ? WHERE prefix = ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(time.Now().Unix(), prefix)
	return err
}

func (d *Database) DeleteFrozen(prefix string) error {
	st, err := d.Db.Prepare("DELETE FROM frozen_prefix WHERE prefix = ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(prefix)
	return err
}

func (d *Database) DeleteFrozenByTime(t *time.Time) error {
	st, err := d.Db.Prepare("DELETE FROM frozen_prefix WHERE created_on <= ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(t.Unix())
	return err
}

func (d *Database) InsertToken(token, fqdn string) (int64, error) {
	st, err := d.Db.Prepare("INSERT INTO token (token, fqdn, created_on) VALUES( ?, ?, ? )")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	resp, err := st.Exec(token, fqdn, time.Now().Unix())
	if err != nil {
		return 0, err
	}

	return resp.LastInsertId()
}

func (d *Database) QueryTokenCount() (int64, error) {
	st, err := d.Db.Prepare("SELECT count(*) FROM token")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	var result int64
	if err := st.QueryRow().Scan(&result); err != nil {
		return 0, err
	}

	return result, nil
}

func (d *Database) QueryTokenByToken(token string) (*model.Token, error) {
	r := &model.Token{}
	st, err := d.Db.Prepare("SELECT * FROM token WHERE token = ?")
	if err != nil {
		return r, err
	}
	defer st.Close()

	if err := st.QueryRow(token).Scan(&r.ID, &r.Token, &r.Fqdn, &r.CreatedOn); err != nil {
		return r, err
	}

	return r, nil
}

func (d *Database) QueryTokenByName(name string) (*model.Token, error) {
	r := &model.Token{}
	st, err := d.Db.Prepare("SELECT * FROM token WHERE fqdn = ?")
	if err != nil {
		return r, err
	}
	defer st.Close()

	if err := st.QueryRow(name).Scan(&r.ID, &r.Token, &r.Fqdn, &r.CreatedOn); err != nil {
		return r, err
	}

	return r, nil
}

func (d *Database) QueryTokenByID(id int64) (*model.Token, error) {
	r := &model.Token{}
	st, err := d.Db.Prepare("SELECT * FROM token WHERE id = ?")
	if err != nil {
		return r, err
	}
	defer st.Close()

	if err := st.QueryRow(id).Scan(&r.ID, &r.Token, &r.Fqdn, &r.CreatedOn); err != nil {
		return r, err
	}

	return r, nil
}

func (d *Database) QueryExpiredTokens(t *time.Time) ([]*model.Token, error) {
	result := make([]*model.Token, 0)
	st, err := d.Db.Prepare("SELECT * FROM token WHERE created_on <= ?")
	if err != nil {
		return result, err
	}
	defer st.Close()

	rows, err := st.Query(t.Unix())
	if err != nil {
		return result, err
	}

	for rows.Next() {
		temp := &model.Token{}
		if err := rows.Scan(&temp.ID, &temp.Token, &temp.Fqdn, &temp.CreatedOn); err != nil {
			return result, err
		}
		result = append(result, temp)
	}

	return result, nil
}

func (d *Database) RenewToken(token string) (int64, int64, error) {
	st, err := d.Db.Prepare("UPDATE token SET created_on = ? WHERE token = ?")
	if err != nil {
		return 0, 0, err
	}
	defer st.Close()

	t := time.Now().Unix()
	resp, err := st.Exec(t, token)
	if err != nil {
		return 0, 0, err
	}

	id, err := resp.LastInsertId()
	if err != nil {
		return 0, 0, err
	}

	return id, t, nil
}

func (d *Database) DeleteToken(token string) error {
	st, err := d.Db.Prepare("DELETE FROM token WHERE token = ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(token)
	return err
}

func (d *Database) InsertRecordA(a *model.RecordA) (int64, error) {
	st, err := d.Db.Prepare("INSERT INTO record_a (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	r, err := st.Exec(a.Fqdn, a.Type, a.Content, a.CreatedOn, a.TID)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *Database) QueryRecordA(name string) (*model.RecordA, error) {
	r := &model.RecordA{}
	st, err := d.Db.Prepare("SELECT * FROM record_a WHERE fqdn = ?")
	if err != nil {
		return r, err
	}
	defer st.Close()

	rows, err := st.Query(name)
	if err != nil {
		return r, err
	}

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.Fqdn, &r.Type, &r.Content, &r.CreatedOn, &r.UpdatedOn, &r.TID); err != nil {
			return r, err
		}
	}

	return r, nil
}

func (d *Database) UpdateRecordA(a *model.RecordA) (int64, error) {
	st, err := d.Db.Prepare("UPDATE record_a SET type = ?, content = ?, created_on = ?, tid = ? WHERE fqdn = ?")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	r, err := st.Exec(a.Type, a.Content, a.CreatedOn, a.TID, a.Fqdn)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *Database) DeleteRecordA(name string) error {
	st, err := d.Db.Prepare("DELETE FROM record_a WHERE fqdn = ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(name)
	return err
}

func (d *Database) InsertSubRecordA(a *model.SubRecordA) (int64, error) {
	st, err := d.Db.Prepare("INSERT INTO sub_record_a (fqdn, type, content, created_on, pid) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	r, err := st.Exec(a.Fqdn, a.Type, a.Content, a.CreatedOn, a.PID)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *Database) UpdateSubRecordA(a *model.SubRecordA) (int64, error) {
	st, err := d.Db.Prepare("UPDATE sub_record_a SET type = ?, content = ?, created_on = ?, pid = ? WHERE fqdn = ?")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	r, err := st.Exec(a.Type, a.Content, a.CreatedOn, a.PID, a.Fqdn)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *Database) QuerySubRecordA(name string) (*model.SubRecordA, error) {
	r := &model.SubRecordA{}
	st, err := d.Db.Prepare("SELECT * FROM sub_record_a WHERE fqdn = ?")
	if err != nil {
		return r, err
	}
	defer st.Close()

	rows, err := st.Query(name)
	if err != nil {
		return r, err
	}

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.Fqdn, &r.Type, &r.Content, &r.CreatedOn, &r.UpdatedOn, &r.PID); err != nil {
			return r, err
		}
	}

	return r, nil
}

func (d *Database) DeleteSubRecordA(name string) error {
	st, err := d.Db.Prepare("DELETE FROM sub_record_a WHERE fqdn = ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(name)
	return err
}

func (d *Database) InsertRecordTXT(a *model.RecordTXT) (int64, error) {
	st, err := d.Db.Prepare("INSERT INTO record_txt (fqdn, type, content, created_on, tid) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	r, err := st.Exec(a.Fqdn, a.Type, a.Content, a.CreatedOn, a.TID)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *Database) UpdateRecordTXT(a *model.RecordTXT) (int64, error) {
	st, err := d.Db.Prepare("UPDATE record_txt SET type = ?, content = ?, created_on = ?, tid = ? WHERE fqdn = ?")
	if err != nil {
		return 0, err
	}
	defer st.Close()

	r, err := st.Exec(a.Type, a.Content, a.CreatedOn, a.TID, a.Fqdn)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d *Database) DeleteRecordTXT(name string) error {
	st, err := d.Db.Prepare("DELETE FROM record_txt WHERE fqdn = ?")
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.Exec(name)
	return err
}

func (d *Database) QueryRecordTXT(name string) (*model.RecordTXT, error) {
	r := &model.RecordTXT{}
	st, err := d.Db.Prepare("SELECT * FROM record_txt WHERE fqdn = ?")
	if err != nil {
		return r, err
	}
	defer st.Close()

	rows, err := st.Query(name)
	if err != nil {
		return r, err
	}

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.Fqdn, &r.Type, &r.Content, &r.CreatedOn, &r.UpdatedOn, &r.TID); err != nil {
			return r, err
		}
	}

	return r, nil
}

func (d *Database) QueryExpiredRecordTXTs(id int64) ([]*model.RecordTXT, error) {
	result := make([]*model.RecordTXT, 0)
	st, err := d.Db.Prepare("SELECT * FROM record_txt WHERE tid = ?")
	if err != nil {
		return result, err
	}
	defer st.Close()

	rows, err := st.Query(id)
	if err != nil {
		return result, err
	}

	for rows.Next() {
		temp := &model.RecordTXT{}
		if err := rows.Scan(&temp.ID, &temp.Fqdn, &temp.Type, &temp.Content, &temp.CreatedOn, &temp.UpdatedOn, &temp.TID); err != nil {
			return result, err
		}
		result = append(result, temp)
	}

	return result, nil
}

func (d *Database) Close() error {
	return d.Db.Close()
}
