package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/joho/godotenv"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/migrations"
	"github.com/tearingItUp786/chatgpt-tui/util"
	"github.com/tearingItUp786/chatgpt-tui/views"
)

var purgeCache bool

func init() {
	flag.BoolVar(&purgeCache, "purge-cache", false, "Invalidate models cache")
}

func main() {
	flag.Parse()

	env := os.Getenv("FOO_ENV")
	if "" == env {
		env = "development"
	}

	godotenv.Load(".env." + env + ".local")
	if "test" != env {
		godotenv.Load(".env.local")
	}
	godotenv.Load(".env." + env)
	godotenv.Load() // The Original .env

	appPath, err := util.GetAppDataPath()
	f, err := tea.LogToFile(filepath.Join(appPath, "debug.log"), "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if "" == apiKey {
		fmt.Println("OPENAI_API_KEY not set; set it in your profile")
		fmt.Printf("export OPENAI_API_KEY=your_key in the config for :%v \n", os.Getenv("SHELL"))
		fmt.Println("Exiting...")
		os.Exit(1)
	}

	// delete files if in dev mode
	util.DeleteFilesIfDevMode()
	// validate config
	configToUse := config.CreateAndValidateConfig()

	// run migrations for our database
	db := util.InitDb()
	err = util.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		log.Println("Error: ", err)
		panic(err)
	}
	defer db.Close()

	if purgeCache {
		err = util.PurgeModelsCache(db)
		if err != nil {
			log.Println("Failed to purge models cache:", err)
		} else {
			log.Println("Models cache invalidated")
		}
	}

	ctx := context.Background()
	ctxWithConfig := config.WithConfig(ctx, &configToUse)

	p := tea.NewProgram(
		views.NewMainView(db, ctxWithConfig),
		tea.WithAltScreen(),
		// tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	_, err = p.Run()
	if err != nil {
		log.Fatal(err)
	}
}
