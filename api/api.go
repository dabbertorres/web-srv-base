package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"webServer/dialogue"
)

type (
	GetDB func(context.Context) (*sql.Conn, error)
	GetSession func(r *http.Request) (dialogue.Conn, error)
)

func Log(logger *log.Logger, r *http.Request, why string) {
	logger.Printf("API %s '%s' (%s, %s): %s\n", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent(), why)
}
