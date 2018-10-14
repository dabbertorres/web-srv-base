package dialogue

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dabbertorres/web-srv-base/logme"
)

const (
	sessionCookie = "session"
)

type sessionCtxKey struct{}

type session struct {
	User       string    `json:"username"`
	IPAddr     string    `json:"ipAddr"`
	Location   string    `json:"location"`
	Expiration time.Time `json:"ttl"`
}

func newSession(w http.ResponseWriter, r *http.Request) (sess session, err error) {
	key, err := genKey(r)
	if err != nil {
		return
	}

	sess.IPAddr = r.RemoteAddr
	sess.Location = r.RequestURI
	sess.Expiration = time.Now().Add(sessionLifetime)

	buf, err := json.Marshal(&sess)
	if err != nil {
		logme.Warn().Printf("marshaling session: %v\n%v\n", err, sess)
		return
	}

	err = db.Put(key, buf, nil)
	if err != nil {
		return
	}

	// delete the session from the db once it hits it's expiration time
	go func(key []byte, exp time.Time) {
		for {
			<-time.After(time.Until(exp))

			// need to make sure the session hasn't been extended

			rawVal, err := db.Get(key, nil)
			if err != nil {
				// already deleted
				return
			}

			var sess session
			err = json.Unmarshal(rawVal, &sess)
			if err != nil {
				// well, it's not useful anyways
				db.Delete(key, nil)
				return
			}

			if sess.Expiration.Before(time.Now()) {
				// it was time!
				db.Delete(key, nil)
				return
			}

			// it's been extended, try again
			exp = sess.Expiration
		}
	}(key, sess.Expiration)

	cookie := &http.Cookie{
		Name:     sessionCookie,
		Value:    string(key),
		MaxAge:   int(sessionLifetime.Seconds()),
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	return
}

func getSession(r *http.Request) (sess session, err error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return
	}

	rawVal, err := db.Get([]byte(cookie.Value), nil)
	if err != nil {
		logme.Info().Println("session does not actually exist")
		return
	}

	err = json.Unmarshal(rawVal, &sess)
	if err != nil {
		logme.Warn().Printf("unmarshaling session: %v\n%s\n", err, string(rawVal))
		return
	}

	return
}

func setSession(r *http.Request, sess *session) (err error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return
	}

	buf, err := json.Marshal(sess)
	if err != nil {
		return
	}

	err = db.Put([]byte(cookie.Value), buf, nil)
	return
}

func delSession(r *http.Request) error {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return err
	}

	return db.Delete([]byte(cookie.Value), nil)
}

func genKey(r *http.Request) (key []byte, err error) {
	const (
		keyRandBytes = 16
	)

	buf := bytes.NewBuffer(nil)
	buf.WriteString(r.RemoteAddr)
	buf.WriteString(r.UserAgent())

	n, err := buf.ReadFrom(io.LimitReader(rand.Reader, keyRandBytes))
	if err != nil {
		return
	}
	if n < keyRandBytes {
		err = errors.New("unable to read enough entropy")
		return
	}

	key = []byte(base64.StdEncoding.EncodeToString(buf.Bytes()))
	return
}
