package main

import (
	"io/ioutil"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/acme/autocert"

	"github.com/dabbertorres/how"
)

const (
	dbPassFile = "/run/secrets/web-srv-db-password"
	certsDir   = "/certs"
	confFile   = "/web.conf"
)

func LoadConfig() (cfg Config, err error) {
	err = how.ParseWithFile(&cfg, confFile)
	if err != nil {
		return
	}

	// get the db password
	var dbPass []byte
	dbPass, err = ioutil.ReadFile(dbPassFile)
	if err != nil {
		return
	}
	cfg.DBAddr = strings.Replace(cfg.DBAddr, "password", strings.TrimSpace(string(dbPass)), -1)

	return
}

func LetsEncryptSetup(cfg *Config) *autocert.Manager {
	return &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		Cache:       autocert.DirCache(certsDir),
		HostPolicy:  autocert.HostWhitelist(cfg.Hostname),
		RenewBefore: time.Duration(cfg.CertRenew) * time.Hour,
		Email:       cfg.CertEmail,
	}
}
