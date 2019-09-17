package routers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rancher/rdns-server/providers"
	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	AuthHeader = "Authorization"
	Bearer     = "Bearer "
)

var (
	excludeAPIMethods = []string{"POST"}
	excludeAPIs       = []string{"domains"}
)

func secure() *mux.Router {
	secure := mux.NewRouter().PathPrefix("/v1").Subrouter().StrictSlash(true)
	secure.Use(tokenMiddleware)
	secure.Methods(http.MethodGet).Path("/domains").HandlerFunc(list)
	secure.Methods(http.MethodGet).Path("/domain/{fqdn}").HandlerFunc(get)
	secure.Methods(http.MethodPost).Path("/domain").HandlerFunc(post)
	secure.Methods(http.MethodPut).Path("/domain/{fqdn}").HandlerFunc(put)
	secure.Methods(http.MethodDelete).Path("/domain/{fqdn}").HandlerFunc(del)
	secure.Methods(http.MethodPut).Path("/domain/{fqdn}/renew").HandlerFunc(renew)
	secure.Methods(http.MethodGet).Path("/domain/{fqdn}/txt").HandlerFunc(getTXT)
	secure.Methods(http.MethodPost).Path("/domain/{fqdn}/txt").HandlerFunc(postTXT)
	secure.Methods(http.MethodPut).Path("/domain/{fqdn}/txt").HandlerFunc(putTXT)
	secure.Methods(http.MethodDelete).Path("/domain/{fqdn}/txt").HandlerFunc(delTXT)
	secure.Methods(http.MethodGet).Path("/domain/{fqdn}/cname").HandlerFunc(getCNAME)
	secure.Methods(http.MethodPost).Path("/domain/{fqdn}/cname").HandlerFunc(postCNAME)
	secure.Methods(http.MethodPut).Path("/domain/{fqdn}/cname").HandlerFunc(putCNAME)
	secure.Methods(http.MethodDelete).Path("/domain/{fqdn}/cname").HandlerFunc(delCNAME)
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

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
	}

	if req.URL.Query().Get("type") == "AAAA" {
		payload.Type = types.RecordTypeAAAA
	} else {
		payload.Type = types.RecordTypeA
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.Get(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("get record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func post(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payload := types.Payload{}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if req.URL.Query().Get("type") == "AAAA" {
		payload.Type = types.RecordTypeAAAA
	} else {
		payload.Type = types.RecordTypeA
	}

	if !completePayload(&payload, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.Post(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("set record(s) failed"))
		return
	}

	output.Token, err = utils.WrapToken(output.Fqdn, output.Token)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("wrap token failed"))
		return
	}

	responseWithData(res, output)
}

func put(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if req.URL.Query().Get("type") == "AAAA" {
		payload.Type = types.RecordTypeAAAA
	} else {
		payload.Type = types.RecordTypeA
	}

	if !completePayload(&payload, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.Put(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("update record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func del(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
	}

	if req.URL.Query().Get("type") == "AAAA" {
		payload.Type = types.RecordTypeAAAA
	} else {
		payload.Type = types.RecordTypeA
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	_, err := p.Delete(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("delete record(s) failed"))
		return
	}

	responseWithNoDatum(res)
}

func renew(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
	}

	if payload.Type == "" {
		if req.URL.Query().Get("type") == "AAAA" {
			payload.Type = types.RecordTypeAAAA
		} else {
			payload.Type = types.RecordTypeA
		}
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.Renew(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("renew record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func getTXT(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeTXT,
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.GetTXT(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("get record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func postTXT(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeTXT,
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !completePayload(&payload, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.PostTXT(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("set record(s) failed"))
		return
	}

	output.Token, err = utils.WrapToken(output.Fqdn, output.Token)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("wrap token failed"))
		return
	}

	responseWithData(res, output)
}

func putTXT(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeTXT,
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !completePayload(&payload, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.PutTXT(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("update record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func delTXT(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeTXT,
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	_, err := p.DeleteTXT(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("delete record(s) failed"))
		return
	}

	responseWithNoDatum(res)
}

func getCNAME(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeCNAME,
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.GetCNAME(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("get record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func postCNAME(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeCNAME,
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !completePayload(&payload, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.PostCNAME(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("set record(s) failed"))
		return
	}

	output.Token, err = utils.WrapToken(output.Fqdn, output.Token)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("wrap token failed"))
		return
	}

	responseWithData(res, output)
}

func putCNAME(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeCNAME,
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, err)
		return
	}

	if !completePayload(&payload, true) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	output, err := p.PutCNAME(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("update record(s) failed"))
		return
	}

	responseWithData(res, output)
}

func delCNAME(res http.ResponseWriter, req *http.Request) {
	p := providers.GetProvider()

	payload := types.Payload{
		Fqdn: mux.Vars(req)["fqdn"],
		Type: types.RecordTypeCNAME,
	}

	if !completePayload(&payload, false) {
		responseWithError(res, http.StatusBadRequest, errors.New("request not valid"))
		return
	}

	_, err := p.DeleteCNAME(payload)
	if err != nil {
		responseWithError(res, http.StatusInternalServerError, errors.New("delete record(s) failed"))
		return
	}

	responseWithNoDatum(res)
}

// completePayload payload completion.
func completePayload(payload *types.Payload, valid bool) bool {
	if valid && payload.Text != "" && (len(payload.Hosts) > 0 || len(payload.SubDomain) > 0) {
		return false
	}

	if valid && payload.Text == "" && len(payload.Hosts) <= 0 && len(payload.SubDomain) <= 0 {
		return false
	}

	if payload.Fqdn != "" && strings.Contains(payload.Fqdn, "*") {
		payload.Wildcard = true
	}

	if payload.Text != "" {
		payload.Type = types.RecordTypeTXT
	}

	if len(payload.Hosts) > 0 {
		payload.Type = utils.HostType(payload.Hosts[0])
	}

	if payload.Text == "" && len(payload.Hosts) <= 0 && len(payload.SubDomain) > 0 {
		for _, v := range payload.SubDomain {
			payload.Type = utils.HostType(v[0])
			break
		}
	}

	return true
}

// tokenMiddleware intercept request.
func tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		intercept := true
		for _, e := range excludeAPIMethods {
			if r.Method == e {
				intercept = false
				break
			}
		}

		for _, e := range excludeAPIs {
			if strings.Contains(r.RequestURI, e) {
				intercept = false
				break
			}
		}

		if intercept {
			domain, ok := mux.Vars(r)["fqdn"]
			if !ok {
				responseWithError(w, http.StatusBadRequest, errors.New("must specific the fqdn"))
				return
			}

			payload := types.Payload{
				Fqdn:     domain,
				Wildcard: strings.Contains(domain, "*"),
			}

			if !utils.CompareToken(strings.TrimLeft(r.Header.Get(AuthHeader), Bearer), payload) {
				responseWithError(w, http.StatusForbidden, errors.New("forbidden to use"))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
