package keepers

import (
	"fmt"
	"strings"
	"time"

	"github.com/rancher/rdns-server/utils"

	"github.com/sirupsen/logrus"
)

const (
	OwnershipPrefix = "ownership-keeper-"
	RotatePrefix    = "rotate-keeper-"
)

var currentKeeper Keeper

type Keeper interface {
	NameCanUse(dnsType, dnsName string) (bool, error)
	SetRotate(dnsType, dnsName string) (Keep, error)
	GetOwnership(dnsType, dnsName string) (Keep, error)
	SetOwnership(dnsType, dnsName string) (Keep, error)
	SetSubOwnership(dnsType, dnsName string, keep Keep) error
	DeleteOwnership(dnsType, dnsName string) error
	RenewOwnership(dnsType, dnsName string, keep Keep) (Keep, error)
	IsSubDomain(dnsType, dnsName, upperType, upperName string) bool
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

// NewTXT use TXT records to maintain ownership concept in provider.
func NewTXT(dnsType, dnsName, token string, expire int64, isOwnership bool) Keep {
	dnsType = strings.ToLower(dnsType)
	dnsName = utils.GetDNSName(dnsName)

	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", expire))
	expires := time.Now().Add(duration)

	keeperName := DNSNameToOwnership(dnsType, dnsName)
	if !isOwnership {
		keeperName = DNSNameToRotate(dnsType, dnsName)
	}

	return Keep{
		Domain: keeperName,
		Type:   dnsType,
		Token:  token,
		Expire: expires.Format(time.RFC3339Nano),
	}
}

// RenewTXT renew expire in rdns-server.
func RenewTXT(keep Keep, expire int64) Keep {
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", expire))
	expires := time.Now().Add(duration)

	keep.Expire = expires.Format(time.RFC3339Nano)
	return keep
}

// ToDNSValue returns dns txt standard value.
func ToDNSValue(k Keep) string {
	value := fmt.Sprintf("%s,%s,%s,%s", k.Domain, k.Type, k.Token, k.Expire)
	return utils.TextWithQuotes(value)
}

func DNSNameToRotate(dnsType, dnsName string) string {
	return strings.ToLower(fmt.Sprintf("%s%s.%s", RotatePrefix, dnsType, dnsName))
}

func DNSNameToOwnership(dnsType, dnsName string) string {
	return strings.ToLower(fmt.Sprintf("%s%s.%s", OwnershipPrefix, dnsType, dnsName))
}
