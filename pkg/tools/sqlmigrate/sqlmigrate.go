package sqlmigrate

import (
	"database/sql"

	"github.com/rancher/rdns-server/pkg/consts"

	"github.com/rubenv/sql-migrate"
)

var (
	downCommand = []string{
		`DROP TABLE IF EXISTS frozen_prefix`,
		`DROP TABLE IF EXISTS record_txt`,
		`DROP TABLE IF EXISTS record_cname`,
		`DROP TABLE IF EXISTS sub_record_a`,
		`DROP TABLE IF EXISTS record_a`,
		`DROP TABLE IF EXISTS token`,
	}
	upCommand = []string{
		`CREATE TABLE IF NOT EXISTS frozen_prefix (
			id INT AUTO_INCREMENT,
			prefix VARCHAR(255) NOT NULL UNIQUE,
			created_on BIGINT NOT NULL,
			PRIMARY KEY (id),
			INDEX index_created_on_frozen (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS token (
			id INT AUTO_INCREMENT,
			token VARCHAR(255) NOT NULL UNIQUE,
			fqdn VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			PRIMARY KEY (id),
			INDEX index_created_on_token (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS record_a (
			id INT AUTO_INCREMENT,
			fqdn VARCHAR(255) NOT NULL UNIQUE,
			type TINYINT NOT NULL,
			content VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			updated_on BIGINT,
			tid INT NOT NULL,
			CONSTRAINT fk_token_a FOREIGN KEY(tid) REFERENCES token(id) ON DELETE CASCADE,
			PRIMARY KEY (id),
			INDEX index_created_on_a (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS sub_record_a (
			id INT AUTO_INCREMENT,
			fqdn VARCHAR(255) NOT NULL UNIQUE,
			type TINYINT NOT NULL,
			content VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			updated_on BIGINT,
			pid INT NOT NULL,
			CONSTRAINT fk_record_a FOREIGN KEY(pid) REFERENCES record_a(id) ON DELETE CASCADE,
			PRIMARY KEY (id),
			INDEX index_created_on_sub_a (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS record_cname (
			id INT AUTO_INCREMENT,
			fqdn VARCHAR(255) NOT NULL UNIQUE,
			type TINYINT NOT NULL,
			content VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			updated_on BIGINT,
			tid INT NOT NULL,
			CONSTRAINT fk_token_cname FOREIGN KEY(tid) REFERENCES token(id) ON DELETE CASCADE,
			PRIMARY KEY (id),
			INDEX index_created_on_cname (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS record_txt (
			id INT AUTO_INCREMENT,
			fqdn VARCHAR(255) NOT NULL UNIQUE,
			type TINYINT NOT NULL,
			content VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			updated_on BIGINT,
			tid INT NOT NULL,
			CONSTRAINT fk_token_txt FOREIGN KEY(tid) REFERENCES token(id) ON DELETE CASCADE,
			PRIMARY KEY (id),
			INDEX index_created_on_txt (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
	}
)

type SQLMigrate struct {
	db     *sql.DB
	source *migrate.MemoryMigrationSource
}

func NewSQLMigrate(db *sql.DB) *SQLMigrate {
	return &SQLMigrate{
		db: db,
		source: &migrate.MemoryMigrationSource{
			Migrations: []*migrate.Migration{
				{
					Id:   "1_initial.sql",
					Down: downCommand,
					Up:   upCommand,
				},
			},
		},
	}
}

func (s *SQLMigrate) Up() (int, error) {
	return migrate.Exec(s.db, consts.DBDriverName, s.source, migrate.Up)
}

func (s *SQLMigrate) Down() (int, error) {
	return migrate.Exec(s.db, consts.DBDriverName, s.source, migrate.Down)
}
