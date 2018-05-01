package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"webServer/api"
	"webServer/api/db"
	"webServer/logme"
)

const (
	// almost RFC3339/ISO8061 - it has no seconds
	reqTimeLayout = "2006-01-02T15:04Z0700"
)

func Visits(r *mux.Route, getDB api.GetDB, getSession api.GetSession) {
	r.Methods(http.MethodGet).Path("/visits").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			dbConn, err := getDB(r.Context())
			if err != nil {
				api.Log(logme.Err(), r, "getting DB connection: "+err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer dbConn.Close()

			sess, err := getSession(r)

			err = CheckLoggedIn(r, dbConn, sess)
			if err != nil {
				if err == ErrNotLoggedIn || err == ErrNotAdmin || err == ErrDisabled {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}

				api.Log(logme.Warn(), r, err.Error())
				return
			}

			start, end, loc, err := visitsParseTimes(r)
			if err != nil {
				api.Log(logme.Err(), r, err.Error())
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			rows, err := dbConn.QueryContext(r.Context(), "select * from visits where time between $1 and $2", start, end)
			if err != nil {
				api.Log(logme.Err(), r, err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			var (
				results []db.Visit
				v       db.Visit
			)
			for rows.Next() {
				rows.Scan(&v.User, &v.Time, &v.IP, &v.UserAgent, &v.Path, &v.Action, &v.Params)
				v.Time = v.Time.In(loc)
				results = append(results, v)
			}

			if rows.Err() != nil {
				api.Log(logme.Err(), r, rows.Err().Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			err = json.NewEncoder(w).Encode(results)
			if err != nil {
				api.Log(logme.Err(), r, err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		})
}

func visitsParseTimes(r *http.Request) (start, end time.Time, loc *time.Location, err error) {
	var (
		startStr = r.FormValue("start")
		endStr   = r.FormValue("end")
	)
	if startStr == "" {
		err = errors.New("did not have start parameter")
		return
	}

	start, err = time.ParseInLocation(reqTimeLayout, startStr, time.UTC)
	if err != nil {
		err = fmt.Errorf("start parameter: %v", err)
		return
	}

	if endStr != "" {
		end, err = time.ParseInLocation(reqTimeLayout, endStr, time.UTC)
		if err != nil {
			err = fmt.Errorf("end parameter: %v", err)
			return
		}
	} else {
		end = time.Now()
	}

	loc = start.Location()
	start = start.UTC()
	end = end.UTC()
	return
}
