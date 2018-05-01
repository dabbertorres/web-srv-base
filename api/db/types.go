package db

import "time"

type (
	User struct {
		Name           string `json:"name"`
		HashedPassword []byte `json:"hashedPassword"`
		Admin          bool
		Enabled        bool
	}

	Method int64

	Visit struct {
		User      string
		Time      time.Time
		IP        string
		UserAgent string
		Path      string
		Action    Method
		Params    string
	}
)

const (
	MethodGet Method = iota
	MethodPost
)
