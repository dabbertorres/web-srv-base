package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/acme/autocert"

	"webServer/how"
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
