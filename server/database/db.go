package database

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
)

type DB struct {
	sql *sql.DB
}
type Tx struct {
	sql *sql.Tx
}

func Open(dataSourceName string) (*DB, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &DB{sql: db}, nil
}

// Begin starts an returns a new transaction.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (db *DB) RowExists(ctx context.Context, query string, args ...interface{}) bool {
	l, _ := icontext.GetLogger(ctx)
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := db.sql.QueryRow(query, args...).Scan(&exists)

	if err != nil && err != sql.ErrNoRows {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("error checking if row exists '%s' %v", args, err)
	}

	return exists
}