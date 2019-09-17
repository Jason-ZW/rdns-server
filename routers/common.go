package routers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rancher/rdns-server/types"
)

func responseWithError(w http.ResponseWriter, httpStatus int, err error) {
	o := types.Response{
		Status:  httpStatus,
		Message: err.Error(),
	}

	res, _ := json.Marshal(o)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(res)
}

func responseWithNoDatum(w http.ResponseWriter) {
	o := types.Response{
		Status: http.StatusOK,
	}

	res, _ := json.Marshal(o)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res)
}

func responseWithData(w http.ResponseWriter, domain types.Domain) {
	if domain.Type == "" {
		domain.Type = types.RecordTypeNone
	}

	o := types.Response{
		Status: http.StatusOK,
		Data:   domain,
		Token:  domain.Token,
	}

	res, err := json.Marshal(o)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res)
}

func responseWithDatum(w http.ResponseWriter, domains []types.Domain) {
	o := types.ResponseList{
		Status: http.StatusOK,
		Datum:  domains,
		Type:   fmt.Sprintf("%s/%s/%s/%s", types.RecordTypeA, types.RecordTypeAAAA, types.RecordTypeTXT, types.RecordTypeCNAME),
	}

	res, err := json.Marshal(o)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res)
}
