package databaseconnectionpool

import (
	"database/sql"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type ConnectionPool struct {
	Db  *sql.DB
	cfg Config
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func New(cfg Config) (pool ConnectionPool, returnErr error) {
	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" ||
		cfg.Password == "" || cfg.Database == "" {
		returnErr = errors.Errorf(
			"All fields must be set (%s)",
			spew.Sdump(cfg))
		return
	}

	pool.cfg = cfg

	db, err := sql.Open("postgres", fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s",
		cfg.User, cfg.Password, cfg.Database, cfg.Host, cfg.Port))
	if err != nil {
		returnErr = errors.Wrapf(err,
			"Couldn't open connection to postgre database (%s)",
			spew.Sdump(cfg))
		return
	}

	if err = db.Ping(); err != nil {
		returnErr = errors.Wrapf(err,
			"Couldn't ping postgre database (%s)",
			spew.Sdump(cfg))
		return
	}

	fmt.Println("Successfully connected to database", cfg.Database)

	pool.Db = db
	return
}

func (pool *ConnectionPool) Close() (returnErr error) {
	if pool.Db == nil {
		return
	}

	if err := pool.Db.Close(); err != nil {
		returnErr = errors.Wrapf(err,
			"Errored closing database connection",
			spew.Sdump(pool.cfg))
	}

	fmt.Println("Successfully closed connection to database", pool.cfg.Database)

	return
}
