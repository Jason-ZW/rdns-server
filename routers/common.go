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

func responseWithDatum(w http.ResponseWriter, datum types.Result) {
	if datum.Type == "" {
		datum.Type = fmt.Sprintf("%s/%s/%s/%s/%s", types.RecordTypeA, types.RecordTypeAAAA, types.RecordTypeTXT, types.RecordTypeCNAME, types.RecordTypeSRV)
	}

	o := types.Response{
		Status: http.StatusOK,
		Datum:  datum,
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
