package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"

	"github.com/gomodule/redigo/redis"
	"golang.org/x/crypto/bcrypt"

	"webServer/dialogue"
	"webServer/logme"
)

type Route interface {
	http.Handler
}

type RouteDB interface {
	Route
	DB() *sql.DB
}

type RouteRedis interface {
	Route
	Redis() *redis.Conn
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

func adminLoginAttempt(db *sql.DB, sessions *dialogue.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logme.Err().Println("Parsing admin login form:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		un := r.Form.Get("username")
		pw := r.Form.Get("password")

		// try to login user with DB

		row := db.QueryRow("select (password, admin, enabled) from users where name = $1", un)

		var (
			hashedPw string
			admin    bool
			enabled  bool
		)
		err := row.Scan(&hashedPw, &admin, &enabled)
		if err != nil {
			logme.Err().Printf("failed admin login as '%s': %v\n", un, err)
			// TODO return bad login page
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(hashedPw), []byte(pw))
		if err != nil {
			logme.Err().Printf("failed admin login as '%s': %v\n", un)
			// TODO return bad login page
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !admin {
			logme.Err().Printf("failed admin login as '%s': not an admin\n", un)
			// TODO return bad login page
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !enabled {
			logme.Err().Printf("failed admin login as '%s': account disabled\n", un)
			// TODO return bad login page
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		logme.Info().Printf("Admin '%s' logged in", un)

		yes, err := sessions.HasSession(r)
		if err != nil {
			logme.Err().Println("checking session validity:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if yes {
			cookie, _ := r.Cookie("session")
			err = sessions.SetUser(r.Context(), dialogue.Key(cookie.Value), un)
			if err != nil {
				logme.Err().Printf("Assigning session to user '%s': %v\n", un, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			key, err := sessions.NewSession(r.Context(), un, r.RemoteAddr, r.UserAgent(), r.URL.Path)
			if err != nil {
				logme.Err().Printf("Creating new session for user '%s': %v\n", un, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: string(key),
			})
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}
