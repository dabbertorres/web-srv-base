package dialogue

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/dabbertorres/web-srv-base/logme"
)

var (
	ErrSessionNotExist = errors.New("session does not exist")
	ErrSessionHasUser  = errors.New("session already has a user")
	ErrAlreadyOpen     = errors.New("db is already open")
)

const (
	dbFilePath = "/sessions/sessions.db"
)

var (
	db              *leveldb.DB
	sessionLifetime time.Duration
)

func Open(lifetime time.Duration) (err error) {
	if db != nil {
		return ErrAlreadyOpen
	}
	sessionLifetime = lifetime
	db, err = leveldb.OpenFile(dbFilePath, &opt.Options{
		Strict: opt.DefaultStrict,
	})
	return err
}

func Close() error {
	return db.Close()
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := getSession(r)
		if err != nil {
			sess, err = newSession(w, r)

			// if we have issues creating sessions, nothing is going to work, so just respond saying we have issues
			if err != nil {
				logme.Err().Println("creating new session:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), sessionCtxKey{}, &sess))
		next.ServeHTTP(w, r)

		setSession(r, &sess)
	})
}

const statsBaseFmt = `sessions leveldb stats:
	Write Delays:         %d
	Write Delay Duration: %s
	Write Paused:         %v
	Alive Snapshots:      %d
	Alive Iterators:      %d
	IO Write:             %d
	IO Read:              %d
	Block Cache:          %d
	Open Tables:          %d
	Levels:`

const statsLevelFmt = `
		Size:      %d
		Tables:    %d
		Reads:     %d
		Writes:    %d
		Durations: %s`

func LogStats(w io.Writer) error {
	var stats leveldb.DBStats
	err := db.Stats(&stats)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, statsBaseFmt,
		stats.WriteDelayCount,
		stats.WriteDelayDuration.String(),
		stats.WritePaused,
		stats.AliveSnapshots,
		stats.AliveIterators,
		stats.IOWrite,
		stats.IORead,
		stats.BlockCacheSize,
		stats.OpenedTablesCount)

	for i := range stats.LevelSizes {
		fmt.Fprintf(w, statsLevelFmt,
			stats.LevelSizes[i],
			stats.LevelTablesCounts[i],
			stats.LevelRead[i],
			stats.LevelWrite[i],
			stats.LevelDurations[i].String())
	}
	fmt.Fprint(w, "\n")

	return nil
}
