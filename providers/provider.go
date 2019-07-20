package providers

import (
	"github.com/rancher/rdns-server/types"

	"github.com/sirupsen/logrus"
)

var currentProvider Provider

type Provider interface {
	List() (types.Result, error)
	Get(payload types.Payload) (types.Result, error)
	Post(payload types.Payload) (types.Result, error)
	Put(payload types.Payload) (types.Result, error)
	Delete(payload types.Payload) (types.Result, error)
	Renew(payload types.Payload) (types.Result, error)
	GetZoneName() string
	GetProviderName() string
}

func SetProvider(p Provider) {
	currentProvider = p
}

func GetProvider() Provider {
	if currentProvider == nil {
		logrus.Fatal("not found any provider")
	}
	return currentProvider
}
