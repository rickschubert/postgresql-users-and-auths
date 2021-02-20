package tables

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rickschubert/postgresql-users-and-auths/databaseconnectionpool"
	"github.com/rickschubert/postgresql-users-and-auths/utils"
)

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

func SetupSessionsTable(dbConnection *databaseconnectionpool.ConnectionPool) SessionsTable {
	sessionsTable, err := NewSessionsTable(SessionsTableConfig{
		ConnectionPool: dbConnection,
	})
	utils.HandleError(err)
	err = sessionsTable.createTable()
	utils.HandleError(err)
	return sessionsTable
}

func CreateSessionForUser(sessionsTable SessionsTable, userId string, active bool) {
	_, err := sessionsTable.InsertSession(userId, active)
	utils.HandleError(err)
}
