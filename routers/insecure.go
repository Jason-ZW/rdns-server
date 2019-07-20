package routers

import (
	"net/http"

	"github.com/gorilla/mux"
)

func insecure() *mux.Router {
	insecure := mux.NewRouter().PathPrefix("/").Subrouter().StrictSlash(true)
	insecure.Methods(http.MethodGet).Path("/healthz").HandlerFunc(healthz)
	return insecure
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
