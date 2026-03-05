package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "./queue.db", "path to sqlite db file")
	file := flag.String("file", "migrations/001_init.sql", "path to migration sql")
	flag.Parse()

	sqlBytes, err := os.ReadFile(*file)
	if err != nil {
		panic(fmt.Errorf("read migration file: %w", err))
	}

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		panic(fmt.Errorf("open db: %w", err))
	}
	defer db.Close()

	if _, err := db.Exec(string(sqlBytes)); err != nil {
		panic(fmt.Errorf("apply migration: %w", err))
	}

	fmt.Println("✅ Migration applied successfully")
}
