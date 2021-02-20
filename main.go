package main

import (
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rickschubert/postgresql-users-and-auths/databaseconnectionpool"
	"github.com/rickschubert/postgresql-users-and-auths/tables"
	"github.com/rickschubert/postgresql-users-and-auths/utils"

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

func connectToDatabase() databaseconnectionpool.ConnectionPool {
	dbConnection, err := databaseconnectionpool.New(databaseconnectionpool.Config{
		Host:     "ec2-52-50-171-4.eu-west-1.compute.amazonaws.com",
		Password: getDatabasePassword(),
		Port:     "5432",
		User:     "hajkxfgyonxjux",
		Database: "d803lv72ks3706",
	})
	utils.HandleError(err)
	return dbConnection
}

func main() {
	loadDotEnvFile()
	dbConnection := connectToDatabase()
	defer dbConnection.Close()

	usersTable := tables.SetupUsersTable(&dbConnection)
	sessionsTable := tables.SetupSessionsTable(&dbConnection)

	username := uuid.NewString()
	newUser := tables.AddNewUserToUsersTable(usersTable, username)
	tables.RetrieveUserByUsername(usersTable, newUser.Username)
	tables.CreateSessionForUser(sessionsTable, newUser.Id, true)
	tables.CreateSessionForUser(sessionsTable, newUser.Id, false)
}
