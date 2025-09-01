package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/Witnot/Gator/internal/config"
	"github.com/Witnot/Gator/internal/database"
)


func main() {
	// Step 1: Read the config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	// Step 2: Open DB connection
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Step 3: Initialize Queries
	dbQueries := database.New(db)

	// Step 4: Store config and db in state
	s := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	// Step 5: Initialize commands struct
	cmds := &commands{handlers: make(map[string]func(*state, command) error)}

	// Step 6: Register command handlers
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))


	// Step 7: Parse CLI args
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: not enough arguments")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	// Create command instance
	cmd := command{name: cmdName, args: cmdArgs}

	// Step 8: Run command
	if err := cmds.run(s, cmd); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

