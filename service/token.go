package service

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/rancher/rdns-server/backend"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func generateToken(fqdn string) (string, error) {
	b := backend.GetBackend()
	origin, err := b.GetToken(fqdn)
	if err != nil {
		logrus.Errorf("failed to get token origin %s, err: %v", fqdn, err)
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(origin), bcrypt.MinCost)
	if err != nil {
		logrus.Errorf("failed to generate token with %s, err: %v", fqdn, err)
		return "", err
	}

	token := base64.StdEncoding.EncodeToString(hash)
	return token, nil
}

func compareToken(fqdn, token string) bool {
	// normal text record & acme text record need special treatment
	fqdnLen := len(strings.Split(fqdn, "."))
	rootDomainLen := len(strings.Split(backend.GetBackend().GetZone(), "."))
	diffLen := fqdnLen - rootDomainLen
	if diffLen > 1 {
		sp := strings.SplitAfterN(fqdn, ".", diffLen)
		fqdn = sp[len(sp)-1]
	}

	hash, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		logrus.Errorf("failed to decode token: %s", fqdn)
		return false
	}

	b := backend.GetBackend()
	origin, err := b.GetToken(fqdn)
	if err != nil {
		logrus.Errorf("failed to get token origin %s, err: %v", fqdn, err)
		return false
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(origin))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"token": token,
			"fqdn":  fqdn,
		}).Errorf("failed to compare token, err: %v", err)
		return false
	}
	logrus.Debugf("token **** matched with fqdn %s", fqdn)
	return true
}

func tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// createDomain and ping and metrics have no need to check token
		logrus.Debugf("request URL path: %s", r.URL.Path)
		if (r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/txt")) ||
			(r.Method != http.MethodPost && !strings.HasPrefix(r.URL.Path, "/ping") && !strings.HasPrefix(r.URL.Path, "/metrics")) {
			authorization := r.Header.Get("Authorization")
			token := strings.TrimLeft(authorization, "Bearer ")
			fqdn, ok := mux.Vars(r)["fqdn"]
			if ok {
				if !compareToken(fqdn, token) {
					returnHTTPError(w, http.StatusForbidden, errors.New("forbidden to use"))
					return
				}
			} else {
				returnHTTPError(w, http.StatusForbidden, errors.New("must specific the fqdn"))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
