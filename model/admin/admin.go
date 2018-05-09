package admin

import (
	"errors"
	"net/http"

	"webServer/db"
	"webServer/dialogue"
	"webServer/logme"
)

var (
	ErrNotAdmin = errors.New("user is not an admin")
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggedIn, username, err := dialogue.IsLoggedIn(r)
		if err != nil {
			logme.Err().Println("checking log-in status:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !loggedIn {
			err = dialogue.SaveLocation(r)
			if err != nil {
				logme.Warn().Println("saving location for session:", err)
			}

			http.Redirect(w, r, "/user/login", http.StatusFound)
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

func Login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logme.Err().Println("parsing user login form:", err)
		return
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")

	admin, err := db.UserIsAdmin(r.Context(), username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logme.Err().Println("checking if user is admin:", err)
		return
	}

	if !admin {
		w.WriteHeader(http.StatusBadRequest)
		logme.Warn().Println("admin login attempt by non-admin:", username)
		return
	}

	canLogin, err := db.UserCanLogin(r.Context(), username, password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logme.Err().Println("checking if admin user can login:", err)
		return
	}

	if !canLogin {
		w.WriteHeader(http.StatusBadRequest)
		logme.Warn().Println("failed admin login attempt for:", username)
		return
	}

	err = dialogue.Login(r, username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logme.Err().Println("assigning admin user to session:", err)
		return
	}

	lastLocation, err := dialogue.GetLastLocation(r)
	if err != nil {
		// this doesn't impact usage, so not an error - will redirect them to the home page
		logme.Warn().Println("getting user's last location:", err)
	}

	http.Redirect(w, r, lastLocation, http.StatusFound)
	return
}
