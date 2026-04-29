package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	service := flag.String("service", "identity", "migration service directory under migrations/")
	command := flag.String("command", "status", "goose command: up, down, status")
	dir := flag.String("dir", "", "migration directory override")
	flag.Parse()

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("POSTGRES_DSN is required")
	}

	migrationDir := *dir
	if migrationDir == "" {
		migrationDir = filepath.Join("migrations", *service)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set goose dialect: %v", err)
	}
	goose.SetTableName("goose_db_version_" + *service)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	switch *command {
	case "up":
		err = goose.Up(db, migrationDir)
	case "down":
		err = goose.Down(db, migrationDir)
	case "status":
		err = goose.Status(db, migrationDir)
	default:
		err = fmt.Errorf("unsupported command %q", *command)
	}
	if err != nil {
		log.Fatal(err)
	}
}
