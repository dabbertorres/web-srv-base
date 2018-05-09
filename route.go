package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"webServer/model/admin"
	"webServer/model/user"
	"webServer/db"
	"webServer/dialogue"
	"webServer/logme"
	"webServer/tmpl"
	"webServer/visitors"
)

func pageHandler(path string, data func(r *http.Request) tmpl.Data) http.HandlerFunc {
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

func RegisterRoutes(router *mux.Router) {
	router.NotFoundHandler = staticFileHandler("app/404.html")

	router.Use(dialogue.Middleware)
	router.Use(db.Middleware)
	router.Use(visitors.Middleware)

	// model paths
	adminRoutes(router.Path("/admin").Subrouter())
	userRoutes(router.Path("/user").Subrouter())

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

func adminRoutes(router *mux.Router) {
	router.Use(admin.Middleware)

	router.Path("/login").
		Methods(http.MethodPost).
		HandlerFunc(admin.Login)

	router.Path("/visits").
		Methods(http.MethodGet).
		HandlerFunc(admin.Visits)
}

func userRoutes(router *mux.Router) {
	router.Use(user.Middleware)

	router.Path("/login").
		Methods(http.MethodPost).
		HandlerFunc(user.Login)
}
