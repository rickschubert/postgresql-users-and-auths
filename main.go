package main

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rickschubert/postgresql-users-and-auths/databaseconnectionpool"

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

func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}

type UsersTable struct {
	connectionPool *databaseconnectionpool.ConnectionPool
}

type UsersTableConfig struct {
	ConnectionPool *databaseconnectionpool.ConnectionPool
}

type UserRow struct {
	Id       string
	Password string
	Username string
}

type SessionsTable struct {
	connectionPool *databaseconnectionpool.ConnectionPool
}

type SessionsTableConfig struct {
	ConnectionPool *databaseconnectionpool.ConnectionPool
}

type SessionRow struct {
	Id     string
	Active bool
	UserId string
}

func NewUsersTable(cfg UsersTableConfig) (table UsersTable, err error) {
	if cfg.ConnectionPool == nil {
		err = errors.New(
			"Can't create table without ConnectionPool instance")
		return
	}

	table.connectionPool = cfg.ConnectionPool
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
	username varchar(100) UNIQUE NOT NULL
)`

	if _, err = table.connectionPool.Db.Exec(qry); err != nil {
		err = errors.Wrapf(err,
			"Users table creation query failed (%s)",
			qry)
		return
	}

	return
}

func NewSessionsTable(cfg SessionsTableConfig) (table SessionsTable, err error) {
	if cfg.ConnectionPool == nil {
		err = errors.New(
			"Can't create table without ConnectionPool instance")
		return
	}

	table.connectionPool = cfg.ConnectionPool
	if err = table.createTable(); err != nil {
		err = errors.Wrapf(err,
			"Couldn't create table during initialization")
		return
	}

	return
}

func (table *SessionsTable) createTable() (err error) {
	const qry = `
CREATE TABLE IF NOT EXISTS sessions (
	id char(36) PRIMARY KEY,
	active  boolean NOT NULL,
	userid char(36) NOT NULL
)`

	if _, err = table.connectionPool.Db.Exec(qry); err != nil {
		err = errors.Wrapf(err,
			"Sessions table creation query failed (%s)",
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

	err = table.connectionPool.Db.
		QueryRow(qry, uuid.NewString(), row.Username, row.Password).
		Scan(&newRow.Id, &newRow.Username, &newRow.Password)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't insert user row into DB (%s)",
			spew.Sdump(row))
		return
	}

	return
}

func (table *SessionsTable) InsertSession(userId string, active bool) (newRow SessionRow, err error) {
	const qry = `
INSERT INTO sessions (
	id,
	active,
	userid
)
VALUES (
	$1,
	$2,
	$3
)
RETURNING
	id, active, userid`

	err = table.connectionPool.Db.
		QueryRow(qry, uuid.NewString(), active, userId).
		Scan(&newRow.Id, &newRow.Active, &newRow.UserId)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't insert session for user %s into DB (%s)",
			userId,
			spew.Sdump(newRow))
		return
	}

	return
}

func (table *UsersTable) GetUserByUsername(username string) (returnRow UserRow, returnErr error) {
	if username == "" {
		returnErr = errors.Errorf("Can't get username with empty string")
		return
	}

	const qry = `
SELECT
	id, username, password
FROM
	users
WHERE
	username = $1`

	iterator, err := table.connectionPool.Db.
		Query(qry, username)
	if err != nil {
		err = errors.Wrapf(err,
			"User listing failed (type=%s)",
			username)
		return
	}
	defer iterator.Close()
	for iterator.Next() {
		var row = UserRow{}
		err = iterator.Scan(
			&row.Id, &row.Username, &row.Password)
		if err != nil {
			err = errors.Wrapf(err,
				"Event row scanning failed (type=%s)",
				username)
			return
		}

		returnRow = row
	}
	if err = iterator.Err(); err != nil {
		returnErr = errors.Wrapf(err,
			"Errored while looping through events listing (type=%s)",
			username)
		return
	}

	return
}

func connectToDatabase() databaseconnectionpool.ConnectionPool {
	dbConnection, err := databaseconnectionpool.New(databaseconnectionpool.Config{
		Host:     "ec2-52-50-171-4.eu-west-1.compute.amazonaws.com",
		Password: getDatabasePassword(),
		Port:     "5432",
		User:     "hajkxfgyonxjux",
		Database: "d803lv72ks3706",
	})
	HandleError(err)
	return dbConnection
}

func setupUsersTable(dbConnection *databaseconnectionpool.ConnectionPool) UsersTable {
	usersTable, err := NewUsersTable(UsersTableConfig{
		ConnectionPool: dbConnection,
	})
	HandleError(err)
	err = usersTable.createTable()
	HandleError(err)
	return usersTable
}

func setupSessionsTable(dbConnection *databaseconnectionpool.ConnectionPool) SessionsTable {
	sessionsTable, err := NewSessionsTable(SessionsTableConfig{
		ConnectionPool: dbConnection,
	})
	HandleError(err)
	err = sessionsTable.createTable()
	HandleError(err)
	return sessionsTable
}

func addNewUserToUsersTable(usersTable UsersTable, username string) UserRow {
	row, err := usersTable.InsertUser(UserRow{
		Password: "thisisthepassword",
		Username: username,
	})
	HandleError(err)
	fmt.Println(spew.Sdump(row))
	return row
}

func retrieveUserByUsername(usersTable UsersTable, username string) {
	row, err := usersTable.GetUserByUsername(username)
	HandleError(err)
	fmt.Println(spew.Sdump(row))
}

func createSessionForUser(sessionsTable SessionsTable, userId string, active bool) {
	_, err := sessionsTable.InsertSession(userId, active)
	HandleError(err)
}

func main() {
	loadDotEnvFile()
	dbConnection := connectToDatabase()
	defer dbConnection.Close()

	usersTable := setupUsersTable(&dbConnection)
	sessionsTable := setupSessionsTable(&dbConnection)

	username := uuid.NewString()
	newUser := addNewUserToUsersTable(usersTable, username)
	retrieveUserByUsername(usersTable, newUser.Username)
	createSessionForUser(sessionsTable, newUser.Id, true)
	createSessionForUser(sessionsTable, newUser.Id, false)
}
