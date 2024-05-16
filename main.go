// Test:
// docker run -p 3306:3306 -e MYSQL_ROOT_PASSWORD=test -e MYSQL_USER=test -e MYSQL_PASSWORD=password -e MYSQL_DATABASE=test mysql
// MYSQL_USER=test MYSQL_PASSWORD=password MYSQL_DB=test go run ./...
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/time/rate"
)

func main() {
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", user, password, database))
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	go func() {
		for range time.Tick(5 * time.Second) {
			stats := db.Stats()
			log.Printf("OpenConnections: %d", stats.OpenConnections)
		}
	}()

	maxConns := 25000
	connCreationRate := 70

	sometimes := rate.Sometimes{
		Interval: time.Duration((1.0/float32(connCreationRate))*1000000) * time.Microsecond,
	}

	log.Printf("Creating %d connections at a rate of %d per second (interval %v)",
		maxConns, connCreationRate, sometimes.Interval,
	)

	var conns []*sql.Conn
	for {
		if len(conns) < maxConns {
			sometimes.Do(func() {
				c, err := db.Conn(context.Background())
				if err != nil {
					log.Printf("Failed to get connection: %v", err)
					return
				}
				conns = append(conns, c)
			})
		}
	}
}
