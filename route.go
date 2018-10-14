package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"github.com/dabbertorres/web-srv-base/db"
	"github.com/dabbertorres/web-srv-base/dialogue"
	"github.com/dabbertorres/web-srv-base/logme"
	"github.com/dabbertorres/web-srv-base/model"
	adminapi "github.com/dabbertorres/web-srv-base/model/admin"
	userapi "github.com/dabbertorres/web-srv-base/model/user"
	"github.com/dabbertorres/web-srv-base/tmpl"
	"github.com/dabbertorres/web-srv-base/view"
	"github.com/dabbertorres/web-srv-base/view/admin"
	"github.com/dabbertorres/web-srv-base/view/user"
	"github.com/dabbertorres/web-srv-base/visitors"
)

func pageHandler(templateName string, builder view.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := builder(r)
		if err != nil {
			logme.Err().Printf("Serving template '%s' for '%s': %v\n", templateName, r.RequestURI, err)

			if buildErr, ok := err.(view.Error); ok {
				w.WriteHeader(buildErr.Status)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		err = tmpl.Build(templateName, w, data)
		if err != nil {
			logme.Err().Printf("Building template '%s' for '%s': %v\n", templateName, r.RequestURI, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func staticFileHandler(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(filepath)
		if err != nil {
			logme.Err().Printf("Serving static file '%s': %v\n", filepath, err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(buf)
		}
	}
}

func RegisterRoutes(router *mux.Router) {
	router.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			err := tmpl.Build("pages/404", w, &view.NotFound{})
			if err != nil {
				logme.Err().Println("Serving 404 page:", err)
			}
			w.WriteHeader(http.StatusNotFound)
		})

	// static content
	for _, base := range []string{"app/content", "app/scripts", "app/style"} {
		pathBase := strings.TrimPrefix(base, "app")
		subR := router.PathPrefix(pathBase).
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

	router.Use(dialogue.Middleware)
	router.Use(db.Middleware)
	router.Use(visitors.Middleware)

	var (
		adminR = router.PathPrefix("/admin").Subrouter()
		userR  = router.PathPrefix("/user").Subrouter()
	)

	adminR.Use(adminapi.Middleware)
	userR.Use(userapi.Middleware)

	baseEndpoints(router)
	adminEndpoints(adminR)
	userEndpoints(userR)

	loginViews(router)
	userViews(userR)
	adminViews(adminR)
}

func baseEndpoints(router *mux.Router) {
	router.Path("/login").
		Methods(http.MethodPost).
		HandlerFunc(model.Login)

	router.Path("/login").
		Methods(http.MethodDelete).
		HandlerFunc(model.Logout)
}

func adminEndpoints(router *mux.Router) {
	router.Path("/visits").
		Methods(http.MethodGet).
		HandlerFunc(adminapi.Visits)
}

func userEndpoints(router *mux.Router) {
	router.Path("/new").
		Methods(http.MethodPost).
		HandlerFunc(userapi.New)
}

func loginViews(router *mux.Router) {
	router.Path("/login").
		Methods(http.MethodGet).
		HandlerFunc(pageHandler("pages/login", view.Login))
}

func userViews(router *mux.Router) {
	router.Path("/profile").
		Methods(http.MethodGet).
		HandlerFunc(user.SelfProfile)

	router.Path("/profile/{username}").
		Methods(http.MethodGet).
		HandlerFunc(user.Profile)
}

func adminViews(router *mux.Router) {
	router.Path("/").
		Methods(http.MethodGet).
		HandlerFunc(pageHandler("pages/admin/dashboard", admin.Dashboard))
}
