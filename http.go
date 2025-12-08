package main

import (
	"net/http"

	"github.com/urfave/negroni"
)

func writeInternalServerErr(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Internal Server Error!"))
}

func writeBadRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 - " + message))
}

func basicAuth(cfg Config) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		user, pass, _ := r.BasicAuth()

		if cfg.Auth.Username != user || cfg.Auth.Password != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		next(w, r)
	})
}
