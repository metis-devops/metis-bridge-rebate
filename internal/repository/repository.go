package repository

import (
	"context"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func Connect(conn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", conn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

type Metis struct {
	db *sqlx.DB
}

func NewMetis(db *sqlx.DB) Metis {
	return Metis{db: db}
}
