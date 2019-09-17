package keepers

import (
	"time"

	"github.com/rancher/rdns-server/types"

	"github.com/sirupsen/logrus"
)

var currentKeeper Keeper

type Keeper interface {
	Close() error
	PrefixCanBeUsed(prefix string) (bool, error)
	IsSubDomain(dnsName string) bool
	GetTokenCount() int64
	DeleteExpiredRotate(t *time.Time) error
	GetExpiredTokens(t *time.Time) ([]Keep, error)
	DeleteExpiredTokens(t *time.Time) error
	SetKeeps(payload types.Payload) (Keep, error)
	PutKeeps(payload types.Payload) (Keep, error)
	GetKeep(payload types.Payload) (Keep, error)
	DeleteKeeps(payload types.Payload) (Keep, error)
	RenewKeeps(payload types.Payload) error
}

type Keep struct {
	ID     string
	Domain string
	Type   string
	Token  string
	Expire *time.Time
}

func SetKeeper(k Keeper) {
	currentKeeper = k
}

func GetKeeper() Keeper {
	if currentKeeper == nil {
		logrus.Fatal("not found any keeper")
	}
	return currentKeeper
}
