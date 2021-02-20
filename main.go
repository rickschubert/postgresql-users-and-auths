package main

import (
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	roach "github.com/rickschubert/postgresql-users-and-auths/roach"

	_ "github.com/lib/pq"
)

func getDatabasePassword() string {
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		log.Fatal("You need to set the DATABASE_PASSWORD environment variable.")
	}
	return password
}

func loadDotEnvFile() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file.")
	}
}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

// EventsTable holds the internals of the table, i.e,
// the manager of this instance's database pool (Roach).
// Here you could also add things like a `logger` with
// some predefined fields (for structured logging with
// context).
type EventsTable struct {
	roach *roach.Roach
}

// EventsTableConfig holds the configuration passed to
// the EventsTable "constructor" (`NewEventsTable`).
type EventsTableConfig struct {
	Roach *roach.Roach
}

// EventRow represents in a `struct` the information we
// can get from the table (some fields are insertable but
// not all - ID and CreatedAt are generated when we `insert`,
// thus, these can only be retrieved).
type EventRow struct {
	Id        string
	Type      string
	CreatedAt time.Time
}

// NewEventsTable creates an instance of EventsTable.
// It performs all of its operation against a pool of connections
// that is managed by `Roach`.
func NewEventsTable(cfg EventsTableConfig) (table EventsTable, err error) {
	if cfg.Roach == nil {
		err = errors.New(
			"Can't create table without Roach instance")
		return
	}

	table.roach = cfg.Roach
	// Always try to create the table just in case we don't
	// create them at the database startup.
	// This won't fail in case the table already exists.
	if err = table.createTable(); err != nil {
		err = errors.Wrapf(err,
			"Couldn't create table during initialization")
		return
	}

	return
}

// createTable tries to create a table. If it already exists or not,
// no error is thrown.
// The operation only fails in case there's a mismatch in table
// definition of if there's a connection error.
func (table *EventsTable) createTable() (err error) {
	const qry = `
CREATE TABLE IF NOT EXISTS events (
	id char(36) PRIMARY KEY,
	type text NOT NULL,
	created_at timestamp with time zone DEFAULT current_timestamp
)`

	// Exec executes a query without returning any rows.
	if _, err = table.roach.Db.Exec(qry); err != nil {
		err = errors.Wrapf(err,
			"Events table creation query failed (%s)",
			qry)
		return
	}

	return
}

func (table *EventsTable) InsertEvent(row EventRow) (newRow EventRow, err error) {
	if row.Type == "" {
		err = errors.Errorf("Can't create event without Type (%s)",
			spew.Sdump(row))
		return
	}

	const qry = `
INSERT INTO events (
	id,
	type
)
VALUES (
	$1,
	$2
)
RETURNING
	id, type`

	// `QueryRow` is a single-row query that, unlike `Query()`, doesn't
	// hold a connection. Errors from `QueryRow` are forwarded to `Scan`
	// where we can get errors from both.
	// Here we perform such query for inserting because we want to grab
	// right from the Database the entry that was inserted (plus the fields
	// that the database generated).
	// If we were just getting a value, we could also check if the query
	// was successfull but returned 0 rows with `if err == sql.ErrNoRows`.
	err = table.roach.Db.
		QueryRow(qry, uuid.NewString(), row.Type).
		Scan(&newRow.Id, &newRow.Type)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't insert user row into DB (%s)",
			spew.Sdump(row))
		return
	}

	return
}

func (table *EventsTable) GetEventsByType(eventType string) (rows []EventRow, err error) {
	if eventType == "" {
		err = errors.Errorf("Can't get event rows with empty type")
		return
	}

	const qry = `
SELECT
	id, type
FROM
	events
WHERE
	type = $1`

	// `Query()` returns an iterator that allows us to fetch rows.
	// Under the hood it prepares the query for us (prepares, executes
	// and then closes the prepared stament). This can be good - less
	// code - and bad - performance-wise. If you aim to reuse a query,
	// multiple times in a method, prepare it once and then use it.
	iterator, err := table.roach.Db.
		Query(qry, eventType)
	if err != nil {
		err = errors.Wrapf(err,
			"Event listing failed (type=%s)",
			eventType)
		return
	}
	// we must explicitly `Close` iterator at the end because the
	// `Query` method reserves a database connection that we can
	// use to fetch data.
	defer iterator.Close()
	// While we don't finish reading the rows from the iterator a
	// connection is kept open for it. If you plan to `break` the
	// loop before the iterator finishes, make sure you call `.Close()`
	// to release the resource (connection). The `defer` statement above
	// would do it at the end of the method but, now you know :)
	for iterator.Next() {
		var row = EventRow{}
		// Here `Scan` performs the data type conversions for us
		// based on the type of the destination variable.
		// If an error occur in the conversion, `Scan` will return
		// that error for you.
		err = iterator.Scan(
			&row.Id, &row.Type)
		if err != nil {
			err = errors.Wrapf(err,
				"Event row scanning failed (type=%s)",
				eventType)
			return
		}

		rows = append(rows, row)
	}
	// If something goes bad during the iteration we would only receive
	// the errors in `iterator.Err()` - an abnormal scenario would call
	// `iterator.Close()` (which would end out loop) and then place the
	// error in iterator. By doing this check we safely know whether we
	// got all our results.
	if err = iterator.Err(); err != nil {
		err = errors.Wrapf(err,
			"Errored while looping through events listing (type=%s)",
			eventType)
		return
	}

	return
}

func main() {
	loadDotEnvFile()
	dbRoach, err := roach.New(roach.Config{
		Host:     "ec2-52-50-171-4.eu-west-1.compute.amazonaws.com",
		Password: getDatabasePassword(),
		Port:     "5432",
		User:     "hajkxfgyonxjux",
		Database: "d803lv72ks3706",
	})
	defer dbRoach.Close()
	CheckError(err)

	eventsTable, err := NewEventsTable(EventsTableConfig{
		Roach: &dbRoach,
	})
	CheckError(err)
	err = eventsTable.createTable()
	CheckError(err)
	row, err := eventsTable.InsertEvent(EventRow{
		Type: "b",
	})
	CheckError(err)
	fmt.Println(spew.Sdump(row))

	rows, err := eventsTable.GetEventsByType("b")
	CheckError(err)
	fmt.Println(spew.Sdump(rows))
}
