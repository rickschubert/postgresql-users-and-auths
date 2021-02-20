package tables

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rickschubert/postgresql-users-and-auths/databaseconnectionpool"
	"github.com/rickschubert/postgresql-users-and-auths/utils"
)

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

func SetupUsersTable(dbConnection *databaseconnectionpool.ConnectionPool) UsersTable {
	usersTable, err := NewUsersTable(UsersTableConfig{
		ConnectionPool: dbConnection,
	})
	utils.HandleError(err)
	err = usersTable.createTable()
	utils.HandleError(err)
	return usersTable
}

func AddNewUserToUsersTable(usersTable UsersTable, username string) UserRow {
	row, err := usersTable.InsertUser(UserRow{
		Password: "thisisthepassword",
		Username: username,
	})
	utils.HandleError(err)
	fmt.Println(spew.Sdump(row))
	return row
}
