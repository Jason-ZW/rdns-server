package routers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rancher/rdns-server/providers"
	"github.com/rancher/rdns-server/types"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	queryParamType     = "type"
	queryParamWildcard = "wildcard"
)

func secure() *mux.Router {
	secure := mux.NewRouter().PathPrefix("/v1").Subrouter().StrictSlash(true)
	secure.Methods(http.MethodGet).Path("/domains").HandlerFunc(list)
	secure.Methods(http.MethodGet).Path("/domain/{domain}").HandlerFunc(get)
	secure.Methods(http.MethodPost).Path("/domain").HandlerFunc(post)
	secure.Methods(http.MethodPut).Path("/domain").HandlerFunc(put)
	secure.Methods(http.MethodDelete).Path("/domain").HandlerFunc(del)
	secure.Methods(http.MethodPut).Path("/renew").HandlerFunc(renew)
	return secure
}

func list(res http.ResponseWriter, _ *http.Request) {
	p := providers.GetProvider()

	output, err := p.List()
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("list record(s) failed"))
		return
	}

	responseWithDatum(res, output)
}

func get(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payloads := types.Payload{
		Domain:   mux.Vars(req)["domain"],
		Type:     req.FormValue(queryParamType),
		Wildcard: req.FormValue(queryParamWildcard) == "true",
	}

	if !isRequestLegal(payloads.Domain, payloads.Wildcard, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not match the wildcard option"))
		return
	}

	output, err := p.Get(payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("get record(s) failed"))
		return
	}

	responseWithDatum(res, output)
}

func post(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payloads := types.Payload{}

	err = json.Unmarshal(body, &payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !isRequestLegal(payloads.Domain, payloads.Wildcard, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not match the wildcard option"))
		return
	}

	output, err := p.Post(payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("set record(s) failed"))
		return
	}

	responseWithDatum(res, output)
}

func put(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payloads := types.Payload{}

	err = json.Unmarshal(body, &payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !isRequestLegal(payloads.Domain, payloads.Wildcard, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not match the wildcard option"))
		return
	}

	output, err := p.Put(payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("update route53 record(s) failed"))
		return
	}

	responseWithDatum(res, output)
}

func del(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payloads := types.Payload{}

	err = json.Unmarshal(body, &payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !isRequestLegal(payloads.Domain, payloads.Wildcard, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not match the wildcard option"))
		return
	}

	_, err = p.Delete(payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("delete route53 record(s) failed"))
		return
	}

	responseWithNoDatum(res)
}

func renew(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payloads := types.Payload{}

	err = json.Unmarshal(body, &payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !isRequestLegal(payloads.Domain, payloads.Wildcard, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not match the wildcard option"))
		return
	}

	output, err := p.Renew(payloads)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("renew route53 record(s) failed"))
		return
	}

	responseWithDatum(res, output)
}

// isRequestLegal filter invalid requests.
func isRequestLegal(dnsName string, wildcard, isPost bool) bool {
	if isPost && dnsName == "" {
		// if dnsName is empty, make request valid.
		return true
	}

	if isPost && wildcard && strings.HasPrefix(dnsName, "*") {
		return true
	}

	if isPost && !wildcard && !strings.HasPrefix(dnsName, "*") {
		// not use wildcard, illegal when dnsName contains "*".
		return true
	}

	if !isPost && dnsName != "" && wildcard && strings.HasPrefix(dnsName, "*") {
		// use wildcard, illegal when dnsName not contains "*".
		return true
	}

	if !isPost && dnsName != "" && !wildcard && !strings.HasPrefix(dnsName, "*") {
		// not use wildcard, illegal when dnsName contains "*".
		return true
	}

	return false
}
