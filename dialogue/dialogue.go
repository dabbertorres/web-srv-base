package dialogue

import (
	"net/http"
	"time"
)

func Login(r *http.Request, user string) (err error) {
	sess, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		err = ErrSessionNotExist
		return
	}

	if sess.User != "" {
		err = ErrSessionHasUser
		return
	}

	sess.User = user
	return
}

func Logout(r *http.Request) (err error) {
	_, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		err = ErrSessionNotExist
		return
	}

	err = delSession(r)
	return
}

func IsLoggedIn(r *http.Request) (loggedIn bool, username string) {
	sess, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		// non-existent session == not logged in
		return
	}

	username = sess.User
	loggedIn = username != ""
	return
}

func SaveLocation(r *http.Request) error {
	sess, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		return ErrSessionNotExist
	}

	sess.Location = r.RequestURI
	return nil
}

func GetLastLocation(r *http.Request) (location string, err error) {
	sess, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		err = ErrSessionNotExist
		return
	}

	location = sess.Location
	return
}

func GetExpiration(r *http.Request) (exp time.Time, err error) {
	sess, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		err = ErrSessionNotExist
		return
	}

	exp = sess.Expiration
	return
}

func ExtendExpiration(r *http.Request) error {
	sess, ok := r.Context().Value(sessionCtxKey{}).(*session)
	if !ok {
		return ErrSessionNotExist
	}

	sess.Expiration = time.Now().Add(sessionLifetime)
	return nil
}
