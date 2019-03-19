package main

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"github.com/tengen-io/server/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

const (
	Password = "hunter2"
)

func MakeTestDb() *sqlx.DB {
	port, err := strconv.Atoi(os.Getenv("TENGEN_DB_PORT"))
	if err != nil {
		log.Fatal("Could not parse TENGEN_DB_PORT: ", err)
	}

	config := &db.PostgresDBConfig{
		Host:     os.Getenv("TENGEN_DB_HOST"),
		Port:     port,
		User:     os.Getenv("TENGEN_DB_USER"),
		Database: os.Getenv("TENGEN_DB_DATABASE"),
		Password: os.Getenv("TENGEN_DB_PASSWORD"),
	}

	db, err := db.NewPostgresDb(config)
	if err != nil {
		log.Fatal("Unable to connect to DB.", err)
	}

	return db
}

func setupFixtures() {
	db := MakeTestDb()
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		panic(err)
	}
	fsrc, err := (&file.File{}).Open("file://db/migrations")
	if err != nil {
		panic(err)
	}
	m, err := migrate.NewWithInstance("file", fsrc, "postgres", driver)
	if err != nil {
		panic(err)
	}
	m.Down()
	err = m.Up()
	if err != nil {
		panic(err)
	}
	writeFixtures()
}

func writeFixtures() {
	db := MakeTestDb()
	hash, _ := bcrypt.GenerateFromPassword([]byte(Password), 4)
	now := pq.FormatTimestamp(time.Now().UTC())
	res := db.QueryRow("INSERT INTO identities (email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id", "test1@tengen.io", hash, now, now)
	var id int64
	res.Scan(&id)
	db.MustExec("INSERT INTO users (identity_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4)", id, "Test User 1", now, now)

	db.MustExec("INSERT INTO games (type, state, board_size, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)", "STANDARD", "INVITATION", 19, now, now)
	db.MustExec("INSERT INTO games (type, state, board_size, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)", "STANDARD", "IN_PROGRESS", 19, now, now)
}

func teardownFixtures() {
	db := MakeTestDb()
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		panic(err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "postgres", driver)
	if err != nil {
		panic(err)
	}
	m.Drop()
}

func TestMain(m *testing.M) {
	godotenv.Load(".env.test")
	setupFixtures()
	defer teardownFixtures()
	rv := m.Run()

	os.Exit(rv)

}
