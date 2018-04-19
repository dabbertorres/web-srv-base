package main

// default config values
const (
	hostname  = "localhost"
	dbDriver  = "mysql"
	dbConn    = "/db"
	redisUrl  = "redis"
	certDir   = "certs/"
	certRenew = 24 * 7
	certEmail = ""
)

type Config struct {
	Hostname      string `how:"hostname,n,specify the hostname name the server should respond as"`
	DBDriver      string `how:"db-driver,,specify the the database driver to use"`
	DBAddr        string `how:"db,,specify the location of the database the server should use"`
	DBPassFile    string `how:"db-password-file,,specify the location of the file containing the db user password"`
	RedisUrl      string `how:"redis,,specify where the redis service is found"`
	RedisPassFile string `how:"redis-password-file,,specify the location of the file containing the redis password"`
	SessionTTL    int    `how:"session-ttl,,specify the time-to-live for a session"`
	CertDir       string `how:"cert-dir,,specify the directory to store HTTPS certs"`
	CertRenew     int    `how:"cert-renew,,specify the number of hours before certs are set to expire to renew certs"`
	CertEmail     string `how:"cert-email,,set a contact email address for Let's Encrypt to send notifications to'"`
}

func DefaultConfig() Config {
	return Config{
		Hostname:      hostname,
		DBDriver:      dbDriver,
		DBAddr:        dbConn,
		DBPassFile:    "",
		RedisUrl:      redisUrl,
		RedisPassFile: "",
		SessionTTL:    0,
		CertDir:       certDir,
		CertRenew:     certRenew,
		CertEmail:     certEmail,
	}
}
