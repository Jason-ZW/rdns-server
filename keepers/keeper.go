package keepers

import (
	"github.com/rancher/rdns-server/types"

	"github.com/sirupsen/logrus"
)

var currentKeeper Keeper

type Keeper interface {
	Close() error
	PrefixCanBeUsed(prefix string) (bool, error)
	SetTransaction(payload types.Payload) (Keep, error)
}

type Keep struct {
	Domain string
	Type   string
	Token  string
	Expire string
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
