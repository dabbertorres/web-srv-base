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

type Conn struct {
	conn redis.Conn
	r    *http.Request
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
			c, err := s.Open(r)
			if err != nil {
				logme.Err().Println("updating session location:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer c.Close()

			err = c.UpdateLocation()
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

func (s *Store) Open(r *http.Request) (Conn, error) {
	c, err := s.pool.GetContext(r.Context())
	return Conn{c, r}, err
}

func (c Conn) SetUser(user string) error {
	key, err := c.r.Cookie("session")
	if err != nil || key == nil {
		return err
	}

	exists, err := redis.Bool(c.conn.Do("exists", key.Value))
	if err != nil {
		return err
	}
	if !exists {
		return ErrSessionNotExist
	}

	setUser, err := redis.Bool(c.conn.Do("hsetnx", key.Value, "user", user))
	if err != nil {
		return err
	}
	if !setUser {
		return ErrSessionHasUser
	}

	return nil
}

func (c Conn) UpdateLocation() error {
	key, err := c.r.Cookie("session")
	if err != nil || key == nil {
		return err
	}

	exists, err := redis.Bool(c.conn.Do("exists", key.Value))
	if err != nil {
		return err
	}
	if !exists {
		return ErrSessionNotExist
	}

	_, err = c.conn.Do("hset", key.Value, "location", c.r.RequestURI)

	return err
}

func (c Conn) Get() (sess Session, err error) {
	var key *http.Cookie
	key, err = c.r.Cookie("session")
	if err != nil || key == nil {
		return
	}

	var reply []interface{}
	reply, err = redis.Values(c.conn.Do("hgetall", key.Value))
	if err != nil {
		return
	}

	err = redis.ScanStruct(reply, &sess)
	if err != nil {
		return
	}

	var ttl int
	ttl, err = redis.Int(c.conn.Do("ttl", key.Value))
	if err != nil {
		return
	}

	sess.TTL = time.Duration(ttl) * time.Second
	return
}

func (c Conn) HasSession() (exists bool, err error) {
	cookie, err := c.r.Cookie("session")
	if err != nil || cookie == nil {
		return false, nil
	}

	exists, err = redis.Bool(c.conn.Do("exists", cookie.Value))
	return
}

func (c Conn) IsLoggedIn() (loggedIn bool, username string, err error) {
	var cookie *http.Cookie

	cookie, err = c.r.Cookie("session")
	if err != nil || cookie == nil {
		return
	}

	username, err = redis.String(c.conn.Do("hget", cookie.Value, "user"))
	if err != nil {
		return
	}

	loggedIn = username != ""
	return
}

func (c Conn) Close() error {
	return c.conn.Close()
}
