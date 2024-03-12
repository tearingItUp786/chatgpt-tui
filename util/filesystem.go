package util

import (
	"log"
	"os"
	"path/filepath"
)

func DeleteFilesIfDevMode() {
	if os.Getenv("DEV_MODE") == "true" {
		// Delete the database file
		appPath, err := GetAppDataPath()
		if err != nil {
			panic(err)
		}
		pathToPersistDb := filepath.Join(appPath, "chat.db")
		err = os.Remove(pathToPersistDb)
		if err != nil {
			log.Println("Error deleting database file:", err)
		}
		// Delete the config file
		pathToPersistedFile := filepath.Join(appPath, "config.json")
		err = os.Remove(pathToPersistedFile)
		if err != nil {
			log.Println("Error deleting config file:", err)
		}
	}
}
