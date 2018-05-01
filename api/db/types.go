package db

import (
	"time"
	"net/http"
)

type (
	User struct {
		Name           string `json:"name"`
		HashedPassword []byte `json:"hashedPassword"`
		Admin          bool   `json:"admin"`
		Enabled        bool   `json:"enabled"`
	}

	Method int64

	Visit struct {
		User      string    `json:"user"`
		Time      time.Time `json:"time"`
		IP        string    `json:"ip"`
		UserAgent string    `json:"userAgent"`
		Path      string    `json:"path"`
		Action    Method    `json:"action"`
		Params    string    `json:"params"`
	}
)

const (
	MethodGet     Method = iota
	MethodHead
	MethodPost
	MethodPut
	MethodPatch
	MethodDelete
	MethodConnect
	MethodOptions
	MethodTrace
)

func (m Method) String() string {
	switch m {
	case MethodGet:
		return http.MethodGet
	case MethodHead:
		return http.MethodHead
	case MethodPost:
		return http.MethodPost
	case MethodPut:
		return http.MethodPut
	case MethodPatch:
		return http.MethodPatch
	case MethodDelete:
		return http.MethodDelete
	case MethodConnect:
		return http.MethodConnect
	case MethodOptions:
		return http.MethodOptions
	case MethodTrace:
		return http.MethodTrace
	default:
		return ""
	}
}
