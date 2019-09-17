package providers

import (
	"github.com/rancher/rdns-server/types"

	"github.com/sirupsen/logrus"
)

var currentProvider Provider

type Provider interface {
	List() ([]types.Domain, error)
	Get(payload types.Payload) (types.Domain, error)
	Post(payload types.Payload) (types.Domain, error)
	Put(payload types.Payload) (types.Domain, error)
	Delete(payload types.Payload) (types.Domain, error)
	Renew(payload types.Payload) (types.Domain, error)
	GetTXT(payload types.Payload) (types.Domain, error)
	PostTXT(payload types.Payload) (types.Domain, error)
	PutTXT(payload types.Payload) (types.Domain, error)
	DeleteTXT(payload types.Payload) (types.Domain, error)
	GetCNAME(payload types.Payload) (types.Domain, error)
	PostCNAME(payload types.Payload) (types.Domain, error)
	PutCNAME(payload types.Payload) (types.Domain, error)
	DeleteCNAME(payload types.Payload) (types.Domain, error)
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
