package dialogue

import (
	"time"
)

type Session struct {
	User     string        `json:"username" redis:"user"`
	IPAddr   string        `json:"ipAddr" redis:"ip"`
	Location string        `json:"location" redis:"location"`
	TTL      time.Duration `json:"ttl"`
}
