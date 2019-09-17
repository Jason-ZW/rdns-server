package routers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func insecure() *mux.Router {
	insecure := mux.NewRouter().PathPrefix("/").Subrouter().StrictSlash(true)
	insecure.Methods(http.MethodGet).Path("/ping").HandlerFunc(ping)
	insecure.Methods(http.MethodGet).Path("/healthz").HandlerFunc(healthz)
	insecure.Methods(http.MethodGet).Path("/metrics").Handler(promhttp.Handler())
	return insecure
}

func ping(w http.ResponseWriter, _ *http.Request) {
	responseWithNoDatum(w)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
