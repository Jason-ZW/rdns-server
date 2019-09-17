package fake

import (
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/types"
)

type Faker struct {
	Token string
}

func NewFaker(token string) *Faker {
	return &Faker{
		Token: token,
	}
}

func (f Faker) Close() error {
	return nil
}

func (f Faker) PrefixCanBeUsed(prefix string) (bool, error) {
	return true, nil
}

func (f Faker) IsSubDomain(dnsName string) bool {
	return true
}

func (f Faker) GetTokenCount() int64 {
	return 0
}

func (f Faker) DeleteExpiredRotate(t *time.Time) error {
	return nil
}

func (f Faker) GetExpiredTokens(t *time.Time) ([]keepers.Keep, error) {
	keeps := make([]keepers.Keep, 0)
	return keeps, nil
}

func (f Faker) DeleteExpiredTokens(t *time.Time) error {
	return nil
}

func (f Faker) SetKeeps(payload types.Payload) (keepers.Keep, error) {
	return keepers.Keep{}, nil
}

func (f Faker) PutKeeps(payload types.Payload) (keepers.Keep, error) {
	return keepers.Keep{}, nil
}

func (f Faker) GetKeep(payload types.Payload) (keepers.Keep, error) {
	return keepers.Keep{
		Domain: payload.Fqdn,
		Type:   payload.Type,
		Token:  f.Token,
	}, nil
}

func (f Faker) DeleteKeeps(payload types.Payload) (keepers.Keep, error) {
	return keepers.Keep{}, nil
}

func (f Faker) RenewKeeps(payload types.Payload) error {
	return nil
}
