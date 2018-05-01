package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"

	"webServer/api/admin"
	"webServer/dialogue"
	"webServer/how"
	"webServer/middleware"
)

type Interrupt struct {
	context.Context
	Cancel context.CancelFunc
}

func SetupInterruptCatch(wait chan<- Interrupt, timeout time.Duration) {
	interrupts := make(chan os.Signal)
	signal.Notify(interrupts, os.Interrupt)

	go func() {
		<-interrupts
		signal.Stop(interrupts)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		wait <- Interrupt{ctx, cancel}
	}()
}

func LoadConfig() (cfg Config, redisPass string, err error) {
	config := how.Config{}
	err = config.Load("webServer", &cfg)
	if err != nil {
		return
	}

	// get the db password
	var dbPass []byte
	dbPass, err = ioutil.ReadFile(cfg.DBPassFile)
	if err != nil {
		return
	}
	cfg.DBAddr = strings.Replace(cfg.DBAddr, "password", strings.TrimSpace(string(dbPass)), -1)

	// get the redis password
	var redisPassBuf []byte
	redisPassBuf, err = ioutil.ReadFile(cfg.RedisPassFile)
	redisPass = strings.TrimSpace(string(redisPassBuf))
	return
}

func SetupRedisPool(cfg *Config, redisPassword string) (pool *redis.Pool, err error) {
	pool = &redis.Pool{
		Dial: func() (conn redis.Conn, err error) {
			return redis.Dial("tcp", cfg.RedisUrl, redis.DialReadTimeout(15*time.Second), redis.DialWriteTimeout(15*time.Second), redis.DialPassword(redisPassword))
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

func SetupDB(cfg *Config) (db *sql.DB, err error) {
	db, err = sql.Open(cfg.DBDriver, cfg.DBAddr+"?parseTime=true")
	if err != nil {
		return
	}

	err = db.Ping()
	if err != nil {
		db.Close()
	}

	return
}

func LetsEncryptSetup(cfg *Config) (man *autocert.Manager, err error) {
	err = os.MkdirAll(cfg.CertDir, 0755)
	if err == nil {
		man = &autocert.Manager{
			Prompt:      autocert.AcceptTOS,
			Cache:       autocert.DirCache(cfg.CertDir),
			HostPolicy:  autocert.HostWhitelist(cfg.Hostname),
			RenewBefore: time.Duration(cfg.CertRenew) * time.Hour,
			Email:       cfg.CertEmail,
			ForceRSA:    false,
		}
	}
	return
}

func RegisterRoutes(r *mux.Router, db *sql.DB, sessions *dialogue.Store) {
	var (
		getDB   = func(ctx context.Context) (*sql.Conn, error) { return db.Conn(ctx) }
		getSess = func(r *http.Request) (dialogue.Conn, error) { return sessions.Open(r) }
	)

	r.NotFoundHandler = staticFileHandler("app/404.html")

	r.Use(sessions.Middleware)
	r.Use(middleware.Visit(getDB, getSess))

	// main
	r.Path("/").
		Methods("GET").
		Handler(staticFileHandler("app/index.html"))

	// admin
	adminR := r.Path("/admin").Subrouter()

	admin.Visits(adminR.NewRoute(), getDB, getSess)

	// TODO admin/dashboard.html
	// TODO admin/login.html

	// style
	r.Path("/style/main.css").
		Methods("GET").
		Handler(staticFileHandler("app/style/main.css"))
}
