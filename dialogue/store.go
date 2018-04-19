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
)

type Key string

type Store struct {
	Lifetime time.Duration `json:"lifetime"`
	Pool     *redis.Pool
}

func NewStore(lifetime time.Duration, pool *redis.Pool) *Store {
	return &Store{
		Lifetime: lifetime,
		Pool:     pool,
	}
}

func (s *Store) NewSession(ctx context.Context, user, ipAddr, userAgent, location string) (key Key, err error) {
	const (
		keyRandBytes = 16
	)

	buf := bytes.NewBuffer(nil)

	buf.WriteString(ipAddr)
	buf.WriteString(userAgent)

	n, err := buf.ReadFrom(io.LimitReader(rand.Reader, keyRandBytes))
	if err != nil {
		return
	}
	if n < keyRandBytes {
		err = errors.New("unable to read enough entropy")
		return
	}

	key = Key(base64.StdEncoding.EncodeToString(buf.Bytes()))

	conn, err := s.Pool.GetContext(ctx)
	if err != nil {
		return
	}
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("exists", key))
	if err != nil {
		return
	}

	if exists {
		err = errors.New("session already exists")
		return
	}

	err = conn.Send("hmset", key, "ip", ipAddr, "location", location)
	if err != nil {
		return
	}

	if user != "" {
		err = conn.Send("hset", key, "user", user)
		if err != nil {
			return
		}
	}

	err = conn.Send("expire", key, int(s.Lifetime.Seconds()))
	if err != nil {
		return
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	return
}

func (s *Store) SetUser(ctx context.Context, key Key, user string) error {
	conn, err := s.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	set, err := redis.Bool(conn.Do("hsetnx", key, "user", user))
	if err != nil {
		return err
	}
	if !set {
		return errors.New("session already has a user")
	}

	return nil
}

func (s *Store) Get(ctx context.Context, key Key) (sess Session, err error) {
	var conn redis.Conn
	conn, err = s.Pool.GetContext(ctx)
	if err != nil {
		return
	}
	defer conn.Close()

	var reply []interface{}
	reply, err = redis.Values(conn.Do("hgetall", key))
	if err != nil {
		return
	}

	err = redis.ScanStruct(reply, &sess)
	if err != nil {
		return
	}

	var ttl int
	ttl, err = redis.Int(conn.Do("ttl", key))
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

	conn, err := s.Pool.GetContext(r.Context())
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
		valid  bool
	)

	cookie, err = r.Cookie("session")
	if err != nil || cookie == nil {
		return
	}

	conn, err = s.Pool.GetContext(r.Context())
	if err != nil {
		return
	}
	defer conn.Close()

	err = conn.Send("exists", cookie.Value)
	if err != nil {
		return
	}

	err = conn.Send("hexists", cookie.Value, "user")
	if err != nil {
		return
	}

	err = conn.Send("hget", cookie.Value, "user")
	if err != nil {
		return
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	valid, err = redis.Bool(conn.Receive())
	if err != nil {
		return
	}
	if !valid {
		err = errors.New("session does not exist")
		return
	}

	loggedIn, err = redis.Bool(conn.Receive())
	if err != nil || !loggedIn {
		return
	}

	username, err = redis.String(conn.Receive())
	return
}
