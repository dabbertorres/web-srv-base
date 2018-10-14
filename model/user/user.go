package user

import (
	"fmt"
	"net/http"

	"github.com/dabbertorres/web-srv-base/db"
	"github.com/dabbertorres/web-srv-base/dialogue"
	"github.com/dabbertorres/web-srv-base/logme"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggedIn, _ := dialogue.IsLoggedIn(r)
		if !loggedIn {
			err := dialogue.SaveLocation(r)
			if err != nil {
				logme.Warn().Println("saving location for session:", err)
			}

			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func New(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logme.Err().Println("parsing user login form:", err)
		// TODO nicer "failed account creation"
		return
	}

	var (
		username        = r.Form.Get("username")
		password        = r.Form.Get("password")
		passwordConfirm = r.Form.Get("passwordConfirm")
	)

	if password != passwordConfirm {
		w.WriteHeader(http.StatusBadRequest)
		// TODO nicer "failed account creation - passwords didn't match"
		return
	}

	err = db.UserNew(r.Context(), username, password, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logme.Err().Println("creating new user:", err)
		return
	}

	// TODO email confirmation of account and all that fun stuff

	fmt.Fprintf(w, "Welcome, %s!", username)
}
