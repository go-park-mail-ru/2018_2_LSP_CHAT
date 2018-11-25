package main

import (
	"database/sql"
	"fmt"
	"os"

	ws "github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "host= " + os.Getenv("DB_HOST") + " user=" + os.Getenv("DB_USER") + " password=" + os.Getenv("DB_PASS") + " dbname=" + os.Getenv("DB_DB") + " sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	defer db.Close()
	ws.Run(":8080", db)
}
