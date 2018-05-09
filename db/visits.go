package db

import (
	"context"
	"database/sql"
	"time"
)

func VisitAdd(ctx context.Context, visit *Visit) (err error) {
	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	_, err = conn.ExecContext(ctx,
		"insert into visits (user, time, ip, userAgent, path, action, params) values (?, ?, ?, ?, ?, ?, ?)",
		visit.User, visit.Time, visit.IP, visit.UserAgent, visit.Path, visit.Method, visit.Params)
	return
}

func VisitsBetween(ctx context.Context, start, end time.Time, location *time.Location) (results []Visit, err error) {
	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	rows, err := conn.QueryContext(ctx, "select * from visits where time between ? and ?", start, end)

	var v Visit
	for rows.Next() {
		rows.Scan(&v.User, &v.Time, &v.IP, &v.UserAgent, &v.Path, &v.Method, &v.Params)
		v.Time = v.Time.In(location)
		results = append(results, v)
	}
	err = rows.Err()

	return
}
