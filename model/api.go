package model

import (
	"log"
	"net/http"

)

func Log(logger *log.Logger, r *http.Request, why string) {
	logger.Printf("API %s '%s' (%s, %s): %s\n", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent(), why)
}
