package utils

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/types"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func Sigterm(done chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	logrus.Info("received SIGTERM. terminating...")
	close(done)
}

func WrapToken(domain, token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	if err != nil {
		logrus.Errorf("failed to generate token with %s, err: %v", domain, err)
		return "", err
	}

	token = base64.StdEncoding.EncodeToString(hash)

	return token, nil
}

func CompareToken(token string, payload types.Payload) bool {
	hash, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		logrus.Errorf("failed to decode token: %s", payload.Fqdn)
		return false
	}

	keep, err := keepers.GetKeeper().GetKeep(payload)
	if err != nil {
		logrus.Errorf("failed to get token: %s", payload.Fqdn)
		return false
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(keep.Token)); err != nil {
		logrus.Errorf("failed to compare token, err: %v", payload.Fqdn)
		return false
	}

	return true
}

func ConvertExpire(create time.Time, ttl int64) *time.Time {
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", ttl))
	e := create.Add(duration)
	return &e
}
