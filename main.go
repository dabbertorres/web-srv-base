package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/gorilla/mux"

	"github.com/dabbertorres/how"
	"github.com/dabbertorres/web-srv-base/db"
	"github.com/dabbertorres/web-srv-base/dialogue"
	"github.com/dabbertorres/web-srv-base/logme"
	"github.com/dabbertorres/web-srv-base/tmpl"
)

func main() {
	exitCode := 0
	defer os.Exit(exitCode)

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	err := logme.Init("logs")
	if err != nil {
		exitCode = 1
		panic(err)
	}
	defer logme.Deinit()

	// config...

	cfg, err := LoadConfig()
	if err != nil {
		if err != how.ErrShowHelp {
			logme.Err().Println("Loading config:", err)
		}
		exitCode = 1
		return
	}

	// state setup...

	err = dialogue.Open(time.Duration(cfg.SessionTTL) * time.Second)
	if err != nil {
		logme.Err().Println("Opening sessions file:", err)
		exitCode = 1
		return
	}
	defer dialogue.Close()

	err = db.Open(cfg.DBAddr, cfg.DBDriver)
	if err != nil {
		logme.Err().Println("Connecting to DB:", err)
		exitCode = 1
		return
	}
	defer db.Close()

	httpsMan := LetsEncryptSetup(&cfg)

	// web interface...

	err = tmpl.Load("app")
	if err != nil {
		logme.Err().Println("Loading templates and pages:", err)
		exitCode = 1
		return
	}

	// run...

	var (
		// handle ACME requests, otherwise redirect all other traffic to the https version
		insecureSrv = startInsecure(httpsMan)
		srv         = startSecure(httpsMan, &cfg)
	)

	// try to shutdown gracefully when signaled...

	<-interrupt
	cancelCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := shutdown(cancelCtx, insecureSrv, srv)

	// wait for shutdown or timeout
	select {
	case <-cancelCtx.Done():
		logme.Err().Println("Shutdown timeout")
		exitCode = 1

	case <-done:
		exitCode = 0
	}
}

func startInsecure(man *autocert.Manager) (srv *http.Server) {
	srv = &http.Server{
		Addr: ":http",
		Handler: man.HTTPHandler(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
			})),
		ErrorLog: logme.Err(),
	}

	// serve http for TLS SNI challenges and redirection to https
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logme.Err().Println("Server (http) ListenAndServe():", err)
		}
	}()
	return
}

func startSecure(man *autocert.Manager, cfg *Config) (srv *http.Server) {
	router := mux.NewRouter().Host(cfg.Hostname).Subrouter()
	RegisterRoutes(router)

	srv = &http.Server{
		Addr:      ":https",
		Handler:   router,
		ErrorLog:  logme.Err(),
		TLSConfig: &tls.Config{GetCertificate: man.GetCertificate},
	}

	go func() {
		err := srv.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			logme.Err().Println("Server (https) ListenAndServeTLS():", err)
		}
	}()
	return
}

func shutdown(ctx context.Context, insecure, secure *http.Server) (<-chan struct{}) {
	wait := sync.WaitGroup{}
	wait.Add(2)

	go func() {
		err := insecure.Shutdown(ctx)
		if err != nil {
			logme.Err().Println("Server (insecure) Shutdown():", err)
		}
		wait.Done()
	}()

	go func() {
		err := secure.Shutdown(ctx)
		if err != nil {
			logme.Err().Println("Server Shutdown():", err)
		}
		wait.Done()
	}()

	done := make(chan struct{})
	go func() {
		wait.Wait()
		close(done)
	}()
	return done
}
