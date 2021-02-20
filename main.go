package main

import (
	"fmt"

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

type UsersTable struct {
	roach *roach.Roach
}

type UsersTableConfig struct {
	Roach *roach.Roach
}

type UserRow struct {
	Id       string
	Password string
	Username string
}

func NewUsersTable(cfg UsersTableConfig) (table UsersTable, err error) {
	if cfg.Roach == nil {
		err = errors.New(
			"Can't create table without Roach instance")
		return
	}

	table.roach = cfg.Roach
	if err = table.createTable(); err != nil {
		err = errors.Wrapf(err,
			"Couldn't create table during initialization")
		return
	}

	return
}

func (table *UsersTable) createTable() (err error) {
	const qry = `
CREATE TABLE IF NOT EXISTS users (
	id char(36) PRIMARY KEY,
	password varchar(64) NOT NULL,
	username varchar(100) NOT NULL
)`

	if _, err = table.roach.Db.Exec(qry); err != nil {
		err = errors.Wrapf(err,
			"Events table creation query failed (%s)",
			qry)
		return
	}

	return
}

func (table *UsersTable) InsertUser(row UserRow) (newRow UserRow, err error) {
	if row.Username == "" || row.Password == "" {
		err = errors.Errorf("Can't create user without username and password (%s)",
			spew.Sdump(row))
		return
	}

	const qry = `
INSERT INTO users (
	id,
	username,
	password
)
VALUES (
	$1,
	$2,
	$3
)
RETURNING
	id, username, password`

	// `QueryRow` is a single-row query that, unlike `Query()`, doesn't
	// hold a connection. Errors from `QueryRow` are forwarded to `Scan`
	// where we can get errors from both.
	// Here we perform such query for inserting because we want to grab
	// right from the Database the entry that was inserted (plus the fields
	// that the database generated).
	// If we were just getting a value, we could also check if the query
	// was successfull but returned 0 rows with `if err == sql.ErrNoRows`.
	err = table.roach.Db.
		QueryRow(qry, uuid.NewString(), row.Username, row.Password).
		Scan(&newRow.Id, &newRow.Username)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't insert user row into DB (%s)",
			spew.Sdump(row))
		return
	}

	return
}

func (table *UsersTable) GetUsersByUsername(username string) (rows []UserRow, err error) {
	if username == "" {
		err = errors.Errorf("Can't get username with empty string")
		return
	}

	const qry = `
SELECT
	id, username, password
FROM
	users
WHERE
	username = $1`

	// `Query()` returns an iterator that allows us to fetch rows.
	// Under the hood it prepares the query for us (prepares, executes
	// and then closes the prepared stament). This can be good - less
	// code - and bad - performance-wise. If you aim to reuse a query,
	// multiple times in a method, prepare it once and then use it.
	iterator, err := table.roach.Db.
		Query(qry, username)
	if err != nil {
		err = errors.Wrapf(err,
			"User listing failed (type=%s)",
			username)
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
		var row = UserRow{}
		// Here `Scan` performs the data type conversions for us
		// based on the type of the destination variable.
		// If an error occur in the conversion, `Scan` will return
		// that error for you.
		err = iterator.Scan(
			&row.Id, &row.Username, &row.Password)
		if err != nil {
			err = errors.Wrapf(err,
				"Event row scanning failed (type=%s)",
				username)
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
			username)
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

	usersTable, err := NewUsersTable(UsersTableConfig{
		Roach: &dbRoach,
	})
	CheckError(err)
	err = usersTable.createTable()
	CheckError(err)
	row, err := usersTable.InsertUser(UserRow{
		Password: "thisisthepassword",
		Username: "thisistheusername",
	})
	CheckError(err)
	fmt.Println(spew.Sdump(row))

	rows, err := usersTable.GetUsersByUsername("b")
	CheckError(err)
	fmt.Println(spew.Sdump(rows))
}
