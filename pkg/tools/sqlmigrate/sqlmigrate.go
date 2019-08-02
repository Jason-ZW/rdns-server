package sqlmigrate

import (
	"database/sql"

	"github.com/rancher/rdns-server/pkg/consts"

	"github.com/rubenv/sql-migrate"
)

var (
	downCommand = []string{
		`DROP TABLE IF EXISTS rdns_prefix`,
		`DROP TABLE IF EXISTS rdns_token`,
		`DROP TABLE IF EXISTS rdns_record`,
		`DROP TABLE IF EXISTS rdns_sub_record`,
	}
	upCommand = []string{
		`CREATE TABLE IF NOT EXISTS rdns_prefix (
			id INT AUTO_INCREMENT,
			prefix VARCHAR(255) NOT NULL UNIQUE,
			created_on BIGINT NOT NULL,
			PRIMARY KEY (id),
			INDEX index_created_on_rdns_prefix (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS rdns_token (
			id INT AUTO_INCREMENT,
			token VARCHAR(255) NOT NULL UNIQUE,
			domain VARCHAR(255) NOT NULL,
			type VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			PRIMARY KEY (id),
			INDEX index_created_on_rdns_token (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS rdns_record (
			id INT AUTO_INCREMENT,
			domain VARCHAR(255) NOT NULL UNIQUE,
			type VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			updated_on BIGINT,
			tid INT NOT NULL,
			CONSTRAINT fk_rdns_token FOREIGN KEY(tid) REFERENCES rdns_token(id) ON DELETE CASCADE,
			PRIMARY KEY (id),
			INDEX index_created_on_rdns_record (created_on)
		) ENGINE=INNODB DEFAULT CHARSET=utf8;`,
		`CREATE TABLE IF NOT EXISTS rdns_sub_record (
			id INT AUTO_INCREMENT,
			domain VARCHAR(255) NOT NULL UNIQUE,
			type VARCHAR(255) NOT NULL,
			created_on BIGINT NOT NULL,
			updated_on BIGINT,
			pid INT NOT NULL,
			CONSTRAINT fk_rdns_record FOREIGN KEY(pid) REFERENCES rdns_ecord(id) ON DELETE CASCADE,
			PRIMARY KEY (id),
			INDEX index_created_on_rdns_sub_record (created_on)
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
					Id:   "init-database",
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
