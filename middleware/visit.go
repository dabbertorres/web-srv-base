package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"webServer/api"
	"webServer/api/db"
	"webServer/logme"
)

func Visit(getDB api.GetDB, getSess api.GetSession) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// middleware should not stop a request from passing through
			defer next.ServeHTTP(w, r)

			sess, err := getSess(r)
			if err != nil {
				logme.Warn().Println("obtaining session handle:", err)
				return
			}

			_, username, err := sess.IsLoggedIn()
			if err != nil {
				logme.Warn().Println("obtaining session user:", err)
			}

			visit := db.Visit{
				// may be empty
				User:      username,
				Time:      time.Now().UTC(),
				IP:        r.RemoteAddr,
				UserAgent: r.UserAgent(),
				Path:      r.RequestURI,
			}

			if r.Method == http.MethodGet {
				visit.Action = db.MethodGet
			} else if r.Method == http.MethodPost {
				visit.Action = db.MethodPost
			}

			r.ParseForm()
			params := bytes.NewBuffer(nil)
			err = json.NewEncoder(params).Encode(r.Form)
			if err != nil {
				logme.Warn().Println("json encoding params:", err)
			}
			visit.Params = params.String()

			conn, err := getDB(r.Context())
			if err != nil {
				logme.Warn().Println("visit middleware:", err)
				return
			}
			defer conn.Close()

			_, err = conn.ExecContext(r.Context(),
				"insert into visits (user, time, ip, userAgent, path, action, params) values (?, ?, ?, ?, ?, ?, ?)",
				visit.User, visit.Time, visit.IP, visit.UserAgent, visit.Path, visit.Action, visit.Params)
			if err != nil {
				logme.Warn().Println("writing visit to db:", err)
			}
		})
	}
}
