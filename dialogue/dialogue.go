package dialogue

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"

	"webServer/logme"
)

var (
	ErrSessionNotExist = errors.New("session does not exist")
	ErrSessionExists   = errors.New("session already exists")
	ErrSessionHasUser  = errors.New("session already has a user")
	ErrNoSessionCookie = errors.New("session cookie is nil")
	ErrNoConn          = errors.New("no redis connection")
)

var (
	pool            *redis.Pool
	sessionLifetime time.Duration
)

func Open(url, password string, lifetime time.Duration) (err error) {
	sessionLifetime = lifetime
	pool = &redis.Pool{
		Dial: func() (conn redis.Conn, err error) {
			return redis.Dial("tcp", url, redis.DialReadTimeout(15*time.Second), redis.DialWriteTimeout(15*time.Second), redis.DialPassword(password))
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) (err error) {
			_, err = conn.Do("ping")
			return
		},
		MaxIdle:     3,
		IdleTimeout: 5 * time.Minute,
	}

	// make sure we have a valid connection
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var testConn redis.Conn
	testConn, err = pool.GetContext(ctx)
	if err != nil {
		pool.Close()
		pool = nil
	} else {
		testConn.Close()
	}

	return
}

func Close() error {
	return pool.Close()
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if we have issues creating sessions, nothing is going to work, so just respond saying we have issues
		err := newSession(w, r)
		if err != nil && err != ErrSessionExists {
			logme.Err().Println("creating new session:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, err := pool.GetContext(r.Context())
		if err != nil {
			logme.Err().Println("getting redis connection:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// not important to functionality, so don't cancel request if it fails
		err = SaveLocation(r)
		if err != nil {
			logme.Warn().Println("updating session location:", err)
		}

		next.ServeHTTP(w, r)
	})
}

func newSession(w http.ResponseWriter, r *http.Request) (err error) {
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

	key := base64.StdEncoding.EncodeToString(buf.Bytes())

	conn, err := pool.GetContext(r.Context())
	if err != nil {
		return
	}
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("exists", key))
	if err != nil {
		return
	}
	if exists {
		err = ErrSessionExists
		return
	}

	err = conn.Send("hmset", key, "ip", r.RemoteAddr, "location", r.RequestURI)
	if err != nil {
		return
	}

	err = conn.Send("expire", key, int(sessionLifetime.Seconds()))
	if err != nil {
		return
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    string(key),
		MaxAge:   int(sessionLifetime.Seconds()),
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	return
}
