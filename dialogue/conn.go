package dialogue

import (
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
)

type connKey struct{}

func Login(r *http.Request, user string) (err error) {
	key, err := r.Cookie("session")
	if err != nil {
		return
	}

	if key == nil {
		err = ErrNoSessionCookie
		return
	}

	conn, ok := r.Context().Value(connKey{}).(redis.Conn)
	if !ok {
		err = ErrNoConn
		return
	}

	exists, err := redis.Bool(conn.Do("exists", key.Value))
	if err != nil {
		return
	}

	if !exists {
		err = ErrSessionNotExist
		return
	}

	setUser, err := redis.Bool(conn.Do("hsetnx", key.Value, "user", user))
	if err != nil {
		return
	}

	if !setUser {
		err = ErrSessionHasUser
	}

	return
}

func IsLoggedIn(r *http.Request) (loggedIn bool, username string, err error) {
	var cookie *http.Cookie

	cookie, err = r.Cookie("session")
	if err != nil || cookie == nil {
		return
	}

	conn, ok := r.Context().Value(connKey{}).(redis.Conn)
	if !ok {
		err = ErrNoConn
		return
	}

	username, err = redis.String(conn.Do("hget", cookie.Value, "user"))
	if err != nil {
		return
	}

	loggedIn = username != ""
	return
}

func SaveLocation(r *http.Request) (err error) {
	key, err := r.Cookie("session")
	if err != nil {
		return
	}

	if key == nil {
		err = ErrNoSessionCookie
		return
	}

	conn, ok := r.Context().Value(connKey{}).(redis.Conn)
	if !ok {
		err = ErrNoConn
		return
	}

	exists, err := redis.Bool(conn.Do("exists", key.Value))
	if err != nil {
		return
	}

	if !exists {
		err = ErrSessionNotExist
		return
	}

	_, err = conn.Do("hset", key.Value, "location", r.RequestURI)

	return err
}

func GetLastLocation(r *http.Request) (location string, err error) {
	key, err := r.Cookie("session")
	if err != nil {
		return
	}

	if key == nil {
		err = ErrNoSessionCookie
		return
	}

	conn, ok := r.Context().Value(connKey{}).(redis.Conn)
	if !ok {
		err = ErrNoConn
		return
	}

	location, err = redis.String(conn.Do("hget", key.Value, "location"))
	return
}

func Get(r *http.Request) (sess Session, err error) {
	var key *http.Cookie
	key, err = r.Cookie("session")
	if err != nil || key == nil {
		return
	}

	conn, ok := r.Context().Value(connKey{}).(redis.Conn)
	if !ok {
		err = ErrNoConn
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
