package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	conn, err := pgx.Connect(context.Background(), "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		tables = append(tables, name)
	}
	if len(tables) > 0 {
		cmd := "DROP TABLE IF EXISTS " + strings.Join(tables, ", ") + " CASCADE"
		fmt.Println(cmd)
		_, err = conn.Exec(context.Background(), cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Drop failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("Dropped all tables.")
}
