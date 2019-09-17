package routers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/rancher/rdns-server/utils"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

func NewRouter(done chan struct{}) {
	recovery := negroni.NewRecovery()
	recovery.Formatter = &negroni.HTMLPanicFormatter{}
	n := negroni.New(
		recovery,
		negroni.NewLogger(),
	)

	router := mux.NewRouter().StrictSlash(true)
	router.PathPrefix("/v1").Handler(n.With(
		negroni.Wrap(secure()),
	))
	router.PathPrefix("/").Handler(n.With(
		negroni.Wrap(insecure()),
	))

	p, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		logrus.WithError(err).Fatalf("failed to convert port flag\n")
	}

	go utils.Sigterm(done)

	go func() {
		addr := fmt.Sprintf("0.0.0.0:%d", p)
		if err := http.ListenAndServe(addr, http.Handler(router)); err != nil {
			logrus.WithError(err).Error("failed to listen and serve http")
			done <- struct{}{}
		}
	}()

	<-done
}
