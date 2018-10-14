package admin

import (
	"net/http"

	"github.com/dabbertorres/web-srv-base/db"
	"github.com/dabbertorres/web-srv-base/dialogue"
	"github.com/dabbertorres/web-srv-base/logme"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggedIn, username := dialogue.IsLoggedIn(r)

		if !loggedIn {
			err := dialogue.SaveLocation(r)
			if err != nil {
				logme.Warn().Println("saving location for session:", err)
			}

			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		admin, err := db.UserIsAdmin(r.Context(), username)
		if err != nil {
			logme.Err().Println("checking if user is an admin:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !admin {
			logme.Warn().Println("non-admin attempt to access admin page by:", username)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
