package util

import (
	"log"
	"os"
)

func Log(msgs ...any) {
	if os.Getenv("DEBUG") == "1" {
		log.Println(msgs...)
	}
}
