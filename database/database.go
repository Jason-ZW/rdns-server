package database

import (
	"time"

	"github.com/rancher/rdns-server/model"

	"github.com/sirupsen/logrus"
)

var currentDatabase Database

type Database interface {
	InsertFrozen(prefix string) error
	QueryFrozen(prefix string) (string, error)
	RenewFrozen(prefix string) error
	DeleteFrozen(prefix string) error
	DeleteFrozenByTime(*time.Time) error
	InsertToken(token, fqdn string) (int64, error)
	QueryTokenCount() (int64, error)
	QueryTokenByToken(token string) (*model.Token, error)
	QueryTokenByName(name string) (*model.Token, error)
	QueryTokenByID(id int64) (*model.Token, error)
	QueryExpiredTokens(*time.Time) ([]*model.Token, error)
	RenewToken(token string) (int64, int64, error)
	DeleteToken(prefix string) error
	InsertRecordA(*model.RecordA) (int64, error)
	UpdateRecordA(*model.RecordA) (int64, error)
	QueryRecordA(name string) (*model.RecordA, error)
	DeleteRecordA(name string) error
	InsertSubRecordA(*model.SubRecordA) (int64, error)
	UpdateSubRecordA(*model.SubRecordA) (int64, error)
	QuerySubRecordA(name string) (*model.SubRecordA, error)
	DeleteSubRecordA(name string) error
	InsertRecordTXT(*model.RecordTXT) (int64, error)
	UpdateRecordTXT(*model.RecordTXT) (int64, error)
	QueryRecordTXT(name string) (*model.RecordTXT, error)
	QueryExpiredRecordTXTs(id int64) ([]*model.RecordTXT, error)
	DeleteRecordTXT(name string) error
	Close() error
}

func SetDatabase(d Database) {
	currentDatabase = d
}

func GetDatabase() Database {
	if currentDatabase == nil {
		logrus.Fatal("not found any database")
	}
	return currentDatabase
}
