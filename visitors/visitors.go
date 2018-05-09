package visitors

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"webServer/db"
	"webServer/dialogue"
	"webServer/logme"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := dialogue.Get(r)
		if err != nil {
			logme.Warn().Println("obtaining session:", err)
		}

		queryParams := r.URL.Query()
		params := bytes.NewBuffer(nil)
		err = json.NewEncoder(params).Encode(queryParams)
		if err != nil {
			logme.Warn().Println("json encoding params:", err)
		}

		visit := &db.Visit{
			User:      sess.User,
			Time:      time.Now().UTC(),
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Path:      r.RequestURI,
			Method:    r.Method,
			Params:    params.String(),
		}

		err = db.VisitAdd(r.Context(), visit)
		if err != nil {
			logme.Warn().Println("writing visit to db:", err)
		}

		next.ServeHTTP(w, r)
	})
}
