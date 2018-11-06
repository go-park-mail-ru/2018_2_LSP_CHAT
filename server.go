package main

import (
	"database/sql"
	"fmt"

	ws "github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "host= localhost user=postgres password=root1 dbname=mytestdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	defer db.Close()
	ws.Run(":8080", db)
}
