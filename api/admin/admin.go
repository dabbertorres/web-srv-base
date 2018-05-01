package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"webServer/dialogue"
)

var (
	ErrNotLoggedIn = errors.New("user is not logged in")
	ErrNotAdmin    = errors.New("user is not an admin")
	ErrDisabled    = errors.New("user is disabled")
	ErrNoSession   = errors.New("user does not have a session")
)

func CheckLoggedIn(r *http.Request, dbConn *sql.Conn, session dialogue.Conn) (err error) {
	var (
		username string
		valid    bool
		admin    bool
		enabled  bool
	)

	valid, username, err = session.IsLoggedIn()
	if err != nil {
		err = fmt.Errorf("failed getting login status: %v", err)
		return
	}

	if !valid {
		err = ErrNotLoggedIn
		return
	}

	err = dbConn.QueryRowContext(r.Context(), "select admin, enabled from users where name = $1", username).Scan(&admin, &enabled)
	if err != nil {
		err = fmt.Errorf("checking if user is an admin: %v", err)
		return
	}

	if !admin {
		err = ErrNotAdmin
		return
	}

	if !enabled {
		err = ErrDisabled
	}

	return
}

func LogIn(r *http.Request, dbConn *sql.Conn, session dialogue.Conn) (err error) {
	err = r.ParseForm()
	if err != nil {
		return
	}

	un := r.Form.Get("username")
	pw := r.Form.Get("password")

	var (
		hashedPw   string
		admin      bool
		enabled    bool
		hasSession bool
	)
	err = dbConn.QueryRowContext(r.Context(), "select password, admin, enabled from users where name = ?", un).Scan(&hashedPw, &admin, &enabled)
	if err != nil {
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPw), []byte(pw))
	if err != nil {
		return
	}

	if !admin {
		err = ErrNotAdmin
		return
	}

	if !enabled {
		err = ErrDisabled
		return
	}

	hasSession, err = session.HasSession()
	if err != nil {
		return
	}

	if hasSession {
		err = session.SetUser(un)
	} else {
		err = ErrNoSession
	}
	return
}
