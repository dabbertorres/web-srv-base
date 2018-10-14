package main

// default config values
const (
	hostname  = "localhost"
	dbDriver  = "mysql"
	dbConn    = "/db"
	certRenew = 24 * 30 // LetsEncrypt recommends renewal at 30 days before expiration for their 90 day certs
	certEmail = ""
)

type Config struct {
	Hostname   string `how-long:"hostname" how-short:"n" how-env:"WEB_SRV_HOST" how-help:"specify the hostname the server should respond as"`
	DBDriver   string `how-long:"db-driver" how-env:"WEB_SRV_DB_DRIVER" how-help:"specify the the database driver to use"`
	DBAddr     string `how-long:"db" how-env:"WEB_SRV_DB" how-help:"specify the location of the database the server should use"`
	SessionTTL int    `how-long:"session-ttl" how-env:"WEB_SRV_SESSION_TTL" how-help:"specify the time-to-live for a session"`
	CertRenew  int    `how-long:"cert-renew" how-env:"WEB_SRV_CERT_RENEW" how-help:"specify the number of hours before certs are set to expire to renew certs"`
	CertEmail  string `how:"cert-email" how-env:"WEB_SRV_CERT_EMAIL" how-help:"set a contact email address for Let's Encrypt to send notifications to'"`
}

func DefaultConfig() Config {
	return Config{
		Hostname:   hostname,
		DBDriver:   dbDriver,
		DBAddr:     dbConn,
		SessionTTL: 0,
		CertRenew:  certRenew,
		CertEmail:  certEmail,
	}
}
