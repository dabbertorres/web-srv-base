package dialogue

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"webServer/logme"

	"github.com/gomodule/redigo/redis"
)

var (
	ErrSessionNotExist = errors.New("session does not exist")
	ErrSessionExists   = errors.New("session already exists")
	ErrSessionHasUser  = errors.New("session already has a user")
)

type Key string

type Store struct {
	lifetime time.Duration
	pool     *redis.Pool
}

func NewStore(lifetime time.Duration, pool *redis.Pool) *Store {
	return &Store{
		lifetime: lifetime,
		pool:     pool,
	}
}

func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if we have issues creating sessions, nothing is going to work, so just respond saying we have issues
		_, err := s.NewSession(w, r)
		if err != nil && err != ErrSessionExists {
			logme.Err().Println("creating new session:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			// not important to functionality, so don't cancel request if it fails
			err = s.UpdateLocation(r, r.RequestURI)
			if err != nil {
				logme.Err().Println("updating session location:", err)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Store) NewSession(w http.ResponseWriter, r *http.Request) (key Key, err error) {
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

	key = Key(base64.StdEncoding.EncodeToString(buf.Bytes()))

	conn, err := s.pool.GetContext(r.Context())
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

	err = conn.Send("expire", key, int(s.lifetime.Seconds()))
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
		MaxAge:   int(s.lifetime.Seconds()),
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	r.AddCookie(cookie)

	return
}

func (s *Store) SetUser(r *http.Request, user string) error {
	conn, err := s.pool.GetContext(r.Context())
	if err != nil {
		return err
	}
	defer conn.Close()

	key, err := r.Cookie("session")
	if err != nil || key == nil {
		return err
	}

	exists, err := redis.Bool(conn.Do("exists", key.Value))
	if err != nil {
		return err
	}
	if !exists {
		return ErrSessionNotExist
	}

	setUser, err := redis.Bool(conn.Do("hsetnx", key.Value, "user", user))
	if err != nil {
		return err
	}
	if !setUser {
		return ErrSessionHasUser
	}

	return nil
}

func (s *Store) UpdateLocation(r *http.Request, location string) error {
	conn, err := s.pool.GetContext(r.Context())
	if err != nil {
		return err
	}
	defer conn.Close()

	key, err := r.Cookie("session")
	if err != nil || key == nil {
		return err
	}

	exists, err := redis.Bool(conn.Do("exists", key.Value))
	if err != nil {
		return err
	}
	if !exists {
		return ErrSessionNotExist
	}

	_, err = conn.Do("hset", key.Value, "location", location)

	return err
}

func (s *Store) Get(r *http.Request) (sess Session, err error) {
	var (
		conn redis.Conn
		key  *http.Cookie
	)
	conn, err = s.pool.GetContext(r.Context())
	if err != nil {
		return
	}
	defer conn.Close()

	key, err = r.Cookie("session")
	if err != nil || key == nil {
		return
	}

	var reply []interface{}
	reply, err = redis.Values(conn.Do("hgetall", key.Value))
	if err != nil {
		return
	}

	err = redis.ScanStruct(reply, &sess)
	if err != nil {
		return
	}

	var ttl int
	ttl, err = redis.Int(conn.Do("ttl", key.Value))
	if err != nil {
		return
	}

	sess.TTL = time.Duration(ttl) * time.Second
	return
}

func (s *Store) HasSession(r *http.Request) (exists bool, err error) {
	cookie, err := r.Cookie("session")
	if err != nil || cookie == nil {
		return false, nil
	}

	conn, err := s.pool.GetContext(r.Context())
	if err != nil {
		return false, err
	}
	defer conn.Close()

	exists, err = redis.Bool(conn.Do("exists", cookie.Value))
	return
}

func (s *Store) IsLoggedIn(r *http.Request) (loggedIn bool, username string, err error) {
	var (
		cookie *http.Cookie
		conn   redis.Conn
	)

	cookie, err = r.Cookie("session")
	if err != nil || cookie == nil {
		return
	}

	conn, err = s.pool.GetContext(r.Context())
	if err != nil {
		return
	}
	defer conn.Close()

	username, err = redis.String(conn.Do("hget", cookie.Value, "user"))
	if err != nil {
		return
	}

	loggedIn = username != ""
	return
}
