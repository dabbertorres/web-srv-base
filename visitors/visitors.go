package visitors

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dabbertorres/web-srv-base/db"
	"github.com/dabbertorres/web-srv-base/dialogue"
	"github.com/dabbertorres/web-srv-base/logme"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, user := dialogue.IsLoggedIn(r)

		queryParams := r.URL.Query()
		params := bytes.NewBuffer(nil)
		err := json.NewEncoder(params).Encode(queryParams)
		if err != nil {
			logme.Warn().Println("json encoding params:", err)
		}

		visit := &db.Visit{
			User:      user,
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
