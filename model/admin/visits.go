package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dabbertorres/web-srv-base/db"
	"github.com/dabbertorres/web-srv-base/logme"
	"github.com/dabbertorres/web-srv-base/model"
)

const (
	// almost RFC3339/ISO8061 - it has no seconds
	reqTimeLayout = "2006-01-02T15:04Z0700"
)

func Visits(w http.ResponseWriter, r *http.Request) {
	start, end, loc, err := visitsParseTimes(r)
	if err != nil {
		model.Log(logme.Err(), r, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	results, err := db.VisitsBetween(r.Context(), start, end, loc)
	if err != nil {
		model.Log(logme.Err(), r, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(results)
	if err != nil {
		model.Log(logme.Err(), r, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
