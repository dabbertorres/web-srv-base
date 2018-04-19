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

	"webServer/dialogue"
	"webServer/how"
	"webServer/logme"
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

func LoadConfig() (cfg Config, redisPass []byte, err error) {
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
	cfg.DBAddr = strings.Replace(cfg.DBAddr, "password", string(dbPass), -1)

	// get the redis password
	redisPass, err = ioutil.ReadFile(cfg.RedisPassFile)
	return
}

func SetupRedisPool(cfg *Config, redisPassword string) (pool *redis.Pool, err error) {
	pool = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", cfg.RedisUrl, redis.DialReadTimeout(5*time.Second), redis.DialWriteTimeout(5*time.Second), redis.DialPassword(redisPassword))
		},
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 5 * time.Minute,
	}

	// make sure we have a valid connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testConn, err := pool.GetContext(ctx)
	if err != nil {
		pool.Close()
	} else {
		testConn.Close()
	}

	return
}

func SetupDB(cfg *Config) (db *sql.DB, err error) {
	db, err = sql.Open(cfg.DBDriver, cfg.DBAddr)
	if err != nil {
		return
	}

	err = db.Ping()
	if err != nil {
		db.Close()
	}

	return
}

func LetsEncryptSetup(cfg *Config) *autocert.Manager {
	return &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		Cache:       autocert.DirCache(cfg.CertDir),
		HostPolicy:  autocert.HostWhitelist(cfg.Hostname),
		RenewBefore: time.Duration(cfg.CertRenew) * time.Hour,
		Email:       cfg.CertEmail,
		ForceRSA:    false,
	}
}

func RegisterRoutes(r *mux.Router, db *sql.DB, sessions *dialogue.Store) {
	r.NotFoundHandler = staticFileHandler("app/404.html")

	// main html
	r.Path("/").
		Methods("GET").
		Handler(staticFileHandler("app/index.html"))

	// admin
	admin := r.Path("/admin").Subrouter()

	// logged-in admin will pass the matcher, going straight to the dashboard
	admin.Methods("GET").
		MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
			ok, err := sessions.HasSession(r)
			if err != nil {
				logme.Err().Println("Route:", rm.Route.GetName(), err)
				return false
			}
			return ok
		}).Handler(staticFileHandler("app/admin/dashboard.html"))

	// not logged in admin will not pass above matcher, going here to login
	admin.Methods("GET").
		Handler(staticFileHandler("app/admin/login.html"))

	admin.Methods("POST").
		Handler(adminLoginAttempt(db, sessions))

	// style
	r.Path("style/main.css").
		Methods("GET").
		Handler(staticFileHandler("app/style/main.style"))
}
