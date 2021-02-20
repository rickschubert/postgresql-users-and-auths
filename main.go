package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func getDatabasePassword() string {
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		log.Fatal("You need to set the DATABASE_PASSWORD environment variable.")
	}
	return password
}

func connectToDatabase() {
	const (
		host   = "ec2-52-50-171-4.eu-west-1.compute.amazonaws.com"
		port   = 5432
		user   = "hajkxfgyonxjux"
		dbname = "d803lv72ks3706"
	)
	fmt.Println("Establishing connection to database.")
	password := getDatabasePassword()
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)
	CheckError(err)
	err = db.Ping()
	CheckError(err)
	fmt.Println(fmt.Sprintf("Connected to database %s", dbname))
}

func loadDotEnvFile() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file.")
	}
}

func main() {
	loadDotEnvFile()
	connectToDatabase()
}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
