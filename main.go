package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"webServer/dialogue"
	"webServer/how"
	"webServer/logme"
)

func main() {
	exitCode := 0
	defer os.Exit(exitCode)

	waitShutdown := make(chan Interrupt)
	SetupInterruptCatch(waitShutdown, time.Second*15)

	err := logme.Init("logs")
	if err != nil {
		panic(err)
	}
	defer logme.Deinit()

	// config...

	cfg, redisPass, err := LoadConfig()
	if err != nil {
		if err != how.ShowingHelp {
			logme.Err().Println("Loading config:", err)
		}
		exitCode = 1
		return
	}

	// state setup...

	redisPool, err := SetupRedisPool(&cfg, string(redisPass))
	if err != nil {
		logme.Err().Println("Connecting to Redis:", err)
		exitCode = 1
		return
	}
	defer redisPool.Close()

	sessions := dialogue.NewStore(time.Hour*24, redisPool)

	db, err := SetupDB(&cfg)
	if err != nil {
		logme.Err().Println("Connecting to DB:", err)
		exitCode = 1
		return
	}
	defer db.Close()

	httpsMan := LetsEncryptSetup(&cfg)

	// web interface...

	insecureSrv := &http.Server{
		Addr:     ":http",
		Handler:  httpsMan.HTTPHandler(nil),
		ErrorLog: logme.Err(),
	}

	router := mux.NewRouter().Host(cfg.Hostname).Subrouter()

	RegisterRoutes(router, db, sessions)

	srv := &http.Server{
		Addr:      ":https",
		Handler:   router,
		ErrorLog:  logme.Err(),
		TLSConfig: &tls.Config{GetCertificate: httpsMan.GetCertificate},
	}

	// serve http for TLS SNI challenges and such. All other requests are redirected to https
	go func() {
		err := insecureSrv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logme.Err().Println("Server (http) ListenAndServe():", err)
		}
	}()

	// actual server! (on https)
	go func() {
		err := srv.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			logme.Err().Println("Server (https) ListenAndServeTLS():", err)
		}
	}()

	interrupt := <-waitShutdown
	defer interrupt.Cancel()

	wait := sync.WaitGroup{}
	wait.Add(2)

	go func() {
		err = insecureSrv.Shutdown(interrupt.Context)
		if err != nil {
			logme.Err().Println("Server (insecure) Shutdown():", err)
		}
		wait.Done()
	}()

	go func() {
		err = srv.Shutdown(interrupt.Context)
		if err != nil {
			logme.Err().Println("Server Shutdown():", err)
		}
		wait.Done()
	}()

	shutdownDone := make(chan struct{}, 2)
	go func() {
		wait.Wait()
		shutdownDone <- struct{}{}
	}()

	// start the timeout sequence
	select {
	case <-interrupt.Done():
		logme.Err().Println("Shutdown timeout")
		exitCode = 1

	case <-shutdownDone:
		exitCode = 0
	}
}
