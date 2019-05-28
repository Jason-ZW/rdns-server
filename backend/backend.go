package backend

import (
	"github.com/rancher/rdns-server/model"

	"github.com/sirupsen/logrus"
)

var currentBackend Backend

type Backend interface {
	Get(opts *model.DomainOptions) (model.Domain, error)
	Set(opts *model.DomainOptions) (model.Domain, error)
	Update(opts *model.DomainOptions) (model.Domain, error)
	Delete(opts *model.DomainOptions) error
	Renew(opts *model.DomainOptions) (model.Domain, error)
	SetText(opts *model.DomainOptions) (model.Domain, error)
	GetText(opts *model.DomainOptions) (model.Domain, error)
	UpdateText(opts *model.DomainOptions) (model.Domain, error)
	DeleteText(opts *model.DomainOptions) error
	GetToken(fqdn string) (string, error)
}

func SetBackend(b Backend) {
	currentBackend = b
}

func GetBackend() Backend {
	if currentBackend == nil {
		logrus.Fatal("Not found any backend")
	}
	return currentBackend
}
