package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"webServer/api/admin"
	"webServer/dialogue"
	"webServer/logme"
	"webServer/middleware"
	"webServer/tmpl"
)

func pageHandler(path string, data func(r *http.Request) interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Build(path, w, data(r))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func staticFileHandler(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(filepath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(buf)
		}
	}
}

func RegisterRoutes(router *mux.Router, db *sql.DB, sessions *dialogue.Store) {
	var (
		getDB   = func(ctx context.Context) (*sql.Conn, error) { return db.Conn(ctx) }
		getSess = func(r *http.Request) (dialogue.Conn, error) { return sessions.Open(r) }
	)

	router.NotFoundHandler = staticFileHandler("app/404.html")

	router.Use(sessions.Middleware)
	router.Use(middleware.Visit(getDB, getSess))

	// api paths
	adminR := router.Path("/admin").Subrouter()

	adminR.Path("/visits").
		Methods(http.MethodGet).
		HandlerFunc(admin.Visits(getDB, getSess))

	// page paths
	pagesR := router.Methods(http.MethodGet).Subrouter()

	for path := range tmpl.Pages() {
		// TODO uh, we need a data() function argument for pageHandler()
		pagesR.Path(path).HandlerFunc(pageHandler(path, nil))
	}

	// static/content paths
	for _, base := range []string{"app/content", "app/scripts", "app/style"} {
		pathBase := strings.TrimPrefix(base, "app")
		subR := router.Path(pathBase).
			Methods(http.MethodGet).
			Subrouter()

		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logme.Err().Printf("creating route for '%s' error: %v\n", path, err)
			} else if info.Mode().IsRegular() {
				routePath := strings.TrimPrefix(path, pathBase)
				routePath = strings.TrimSuffix(routePath, filepath.Ext(routePath))

				subR.Path(routePath).HandlerFunc(staticFileHandler(path))
			}
			return nil
		})
	}
}
