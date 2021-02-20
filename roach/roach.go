package roach

import (
	"database/sql"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// Roach holds the connection pool to the database - created by a configuration
// object (`Config`).
type Roach struct {
	// Db holds a sql.DB pointer that represents a pool of zero or more
	// underlying connections - safe for concurrent use by multiple
	// goroutines -, with freeing/creation of new connections all managed
	// by `sql/database` package.
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

func New(cfg Config) (roach Roach, returnErr error) {
	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" ||
		cfg.Password == "" || cfg.Database == "" {
		returnErr = errors.Errorf(
			"All fields must be set (%s)",
			spew.Sdump(cfg))
		return
	}

	roach.cfg = cfg

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

	roach.Db = db
	return
}

func (r *Roach) Close() (returnErr error) {
	if r.Db == nil {
		return
	}

	if err := r.Db.Close(); err != nil {
		returnErr = errors.Wrapf(err,
			"Errored closing database connection",
			spew.Sdump(r.cfg))
	}

	fmt.Println("Successfully closed connection to database", r.cfg.Database)

	return
}
