PostgresQL table creation and row inserts with Golang
=====================================================

This is a small learning project I created for myself in order to get comfortable with working with PostgresQL in conjunction. I have a free PostgresQL database created in Heroku (hosted in AWS) and this project does the following:

- Creates a users table if not present
- Creates a sessions table if not present
- Creates a new user with a UUID as username and a fixed password
- Creates two new sessions associated with the ID of the user, one active and one non-active one

Given this simple setup it would now be possible to create a small REST API to create a signup and login flow.

# Tools used

This project is heavily influenced by [a blog post created by user beld](https://medium.com/@beld_pro/postgres-with-golang-3b788d86f2ef). The blog post uses the [lib-pg](https://pkg.go.dev/github.com/lib/pq@v1.9.0) driver to establish a database connection to PostgresQL; all database manipulations are done using the standard `database/sql` package though. Another package this blog post introduced me to is [spew](https://pkg.go.dev/github.com/davecgh/go-spew@v1.1.1/spew) which is very useful for verbose logging. On top of that I use [google/uuid](https://pkg.go.dev/github.com/google/uuid@v1.2.0) to create unique IDs and the [godotenv](https://pkg.go.dev/github.com/joho/godotenv@v1.3.0) package to read the postgresQL password from a local `.env` file.

# Development notes
- Environment variables can be placed in a `.env` file