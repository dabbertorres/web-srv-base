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
				logme.Warn().Println("obtaining session connection:", err)
				return
			}
			defer sess.Close()

			_, username, err := sess.IsLoggedIn()
			if err != nil {
				logme.Warn().Println("obtaining session user:", err)
			}

			queryParams := r.URL.Query()
			params := bytes.NewBuffer(nil)
			err = json.NewEncoder(params).Encode(queryParams)
			if err != nil {
				logme.Warn().Println("json encoding params:", err)
			}

			visit := db.Visit{
				// may be empty
				User:      username,
				Time:      time.Now().UTC(),
				IP:        r.RemoteAddr,
				UserAgent: r.UserAgent(),
				Path:      r.RequestURI,
				Method:    r.Method,
				Params:    params.String(),
			}

			conn, err := getDB(r.Context())
			if err != nil {
				logme.Warn().Println("obtaining DB connection:", err)
				return
			}
			defer conn.Close()

			_, err = conn.ExecContext(r.Context(),
				"insert into visits (user, time, ip, userAgent, path, action, params) values (?, ?, ?, ?, ?, ?, ?)",
				visit.User, visit.Time, visit.IP, visit.UserAgent, visit.Path, visit.Method, visit.Params)
			if err != nil {
				logme.Warn().Println("writing visit to db:", err)
			}
		})
	}
}
