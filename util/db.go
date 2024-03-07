package util

import (
	"database/sql"
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func GetAppDataPath() (string, error) {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Define the application-specific part of the path
	appDirName := ".chatgpt-tui"

	// Combine them to form the full path
	fullPath := filepath.Join(homeDir, appDirName)

	// Optionally, create the directory if it doesn't already exist
	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}

//go:embed chat.db
var dbEmbed embed.FS

func InitDb() *sql.DB {
	appPath, err := GetAppDataPath()
	if err != nil {
		panic(err)
	}

	pathToPersistDb := filepath.Join(appPath, "chat.db")

	// Check if the database file already exists at the persistent location
	if _, err := os.Stat(pathToPersistDb); os.IsNotExist(err) {
		// The database does not exist, extract from embedded
		dbFile, err := dbEmbed.Open("chat.db")
		if err != nil {
			panic(err)
		}
		defer dbFile.Close()

		// Ensure the directory exists
		if err := os.MkdirAll(filepath.Dir(pathToPersistDb), 0755); err != nil {
			panic(err)
		}

		// Create the persistent file
		outFile, err := os.Create(pathToPersistDb)
		if err != nil {
			panic(err)
		}
		defer outFile.Close()

		// Copy the embedded database to the persistent file
		if _, err := io.Copy(outFile, dbFile); err != nil {
			panic(err)
		}
	} else if err != nil {
		// An error occurred checking for the file, unrelated to file existence
		panic(err)
	}

	// Open the database from the persistent location
	db, err := sql.Open("sqlite3", pathToPersistDb)
	if err != nil {
		panic(err)
	}

	// Apply migrations here as necessary
	// This is a placeholder. Replace with your actual migration logic.

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
		log.Println(err)
		panic(err)
	}
}
