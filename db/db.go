package db

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"github.com/dabbertorres/web-srv-base/logme"
)

type connKey struct{}

var (
	ErrNoDB = errors.New("no db connection")
	handle  *sql.DB
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := handle.Conn(r.Context())
		if err != nil {
			logme.Err().Println("error getting db connection:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		ctx := context.WithValue(r.Context(), connKey{}, conn)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Open(dbAddr, driver string) (err error) {
	handle, err = sql.Open(driver, dbAddr+"?parseTime=true")
	if err != nil {
		return
	}

	err = handle.Ping()
	if err != nil {
		handle.Close()
	}

	return
}

func Close() (err error) {
	if handle != nil {
		err = handle.Close()
	}
	return
}
