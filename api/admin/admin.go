package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"webServer/dialogue"
)

var (
	ErrNotLoggedIn = errors.New("user is not logged in")
	ErrNotAdmin    = errors.New("user is not an admin")
	ErrDisabled    = errors.New("user is disabled")
)

func CheckLoggedIn(r *http.Request, dbConn *sql.Conn, sessions *dialogue.Store) (err error) {
	var (
		username string
		valid    bool
		admin    bool
		enabled  bool
	)

	valid, username, err = sessions.IsLoggedIn(r)
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
