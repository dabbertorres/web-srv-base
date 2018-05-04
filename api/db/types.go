package db

import (
	"time"
)

type (
	User struct {
		Name           string `json:"name"`
		HashedPassword []byte `json:"hashedPassword"`
		Admin          bool   `json:"admin"`
		Enabled        bool   `json:"enabled"`
	}

	Visit struct {
		User      string    `json:"user"`
		Time      time.Time `json:"time"`
		IP        string    `json:"ip"`
		UserAgent string    `json:"userAgent"`
		Path      string    `json:"path"`
		Method    string    `json:"action"`
		Params    string    `json:"params"`
	}
)
