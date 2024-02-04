package util

import (
	"database/sql"
	"io/fs"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func InitDb() *sql.DB {
	db, err := sql.Open("sqlite3", "./foo.db")
	checkErr(err)
	return db
}

func migrate(db *sql.DB, dir string) error {
	err := goose.SetDialect("sqlite3")
	if err != nil {
		log.Printf("migrate: %v", err)
		return err
	}
	err = goose.Up(db, dir)
	if err != nil {
		log.Printf("migrate: %v", err)
		return err
	}
	return nil
}

func MigrateFS(db *sql.DB, migrationsFS fs.FS, dir string) error {
	if dir == "" {
		dir = "."
	}
	goose.SetBaseFS(migrationsFS)
	defer func() {
		goose.SetBaseFS(nil)
	}()
	return migrate(db, dir)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
