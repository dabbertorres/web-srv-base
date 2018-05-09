package db

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExist              = errors.New("user already exists")
	ErrUserDisabledOrNotExist = errors.New("user is disabled, or does not exist")
)

func UserNew(ctx context.Context, username, password string, admin bool) (err error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	result, err := conn.ExecContext(ctx, "insert into users (name, password, admin, enabled) values (?, ?, ?, ?)", username, hashed, admin, true)
	if err != nil {
		return
	}

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		err = ErrUserExist
	}

	return
}

func UserCanLogin(ctx context.Context, username, password string) (can bool, err error) {
	var hashed []byte

	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	err = conn.QueryRowContext(ctx, "select password from users where name = ? and enabled = true", username).Scan(&hashed)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrUserDisabledOrNotExist
		}
		return
	}

	err = bcrypt.CompareHashAndPassword(hashed, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			err = nil
		}
	} else {
		can = true
	}

	return
}

func UserSetEnabled(ctx context.Context, username string, enabled bool) (err error) {
	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	result, err := conn.ExecContext(ctx, "update users set enabled = ? where name = ?", enabled, username)
	if err != nil {
		return
	}

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		err = ErrUserDisabledOrNotExist
	}

	return
}

func UserChangePassword(ctx context.Context, username, newPassword string) (err error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	result, err := conn.ExecContext(ctx, "update users set password = ? where name = ?", hashed, username)
	if err != nil {
		return
	}

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		err = ErrUserExist
	}

	return
}

func UserIsAdmin(ctx context.Context, username string) (admin bool, err error) {
	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	err = conn.QueryRowContext(ctx, "select admin from users where name = ?", username).Scan(&admin)
	return
}

func UserIsEnabled(ctx context.Context, username string) (yes bool, err error) {
	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	err = conn.QueryRowContext(ctx, "select enabled from users where name = ?", username).Scan(&yes)
	return
}

func UserExists(ctx context.Context, username string) (yes bool, err error) {
	var name string

	conn, ok := ctx.Value(connKey{}).(*sql.Conn)
	if !ok {
		err = ErrNoDB
		return
	}

	err = conn.QueryRowContext(ctx, "select name from users where name = ?", username).Scan(&name)
	yes = err != sql.ErrNoRows
	return
}
