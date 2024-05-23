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
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/time/rate"
)

func main() {
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	host := os.Getenv("MYSQL_HOST")
	database := os.Getenv("MYSQL_DATABASE")

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", user, password, host, database))
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	maxConnsEnv := os.Getenv("MAX_CONNS")
	var maxConns int
	if maxConnsEnv == "" {
		maxConns = 1000
	} else {
		maxConns, err = strconv.Atoi(maxConnsEnv)
		if err != nil {
			log.Fatalf("Invalid value '%s' for MAX_CONNS: %v", maxConnsEnv, err)
		}
	}

	connCreationRateEnv := os.Getenv("CONN_CREATION_RATE")
	var connCreationRate int
	if connCreationRateEnv == "" {
		connCreationRate = 1
	} else {
		connCreationRate, err = strconv.Atoi(connCreationRateEnv)
		if err != nil {
			log.Fatalf("Invalid value '%s' for CONN_CREATION_RATE: %v", connCreationRateEnv, err)
		}
	}

	sometimes := rate.Sometimes{
		Interval: time.Duration((1.0/float32(connCreationRate))*1000000) * time.Microsecond,
	}

	log.Printf("Creating %d connections at a rate of %d per second (interval %v)",
		maxConns, connCreationRate, sometimes.Interval,
	)

	timeSpentCreatingConns := new(int)

	go func() {
		start := time.Now()
		for range time.Tick(5 * time.Second) {
			stats := db.Stats()
			elapsed := time.Since(start)
			log.Printf(
				"OpenConnections: %d; AverageConnCreationTime: %d; CreationRate: %d",
				stats.OpenConnections,
				*timeSpentCreatingConns/stats.OpenConnections,
				stats.OpenConnections/int(elapsed.Seconds()),
			)
		}
	}()

	var conns []*sql.Conn
	for {
		if len(conns) < maxConns {
			sometimes.Do(func() {
				start := time.Now()
				c, err := db.Conn(context.Background())
				if err != nil {
					log.Printf("Failed to get connection: %v", err)
					return
				}
				*timeSpentCreatingConns += int(time.Since(start).Milliseconds())

				conns = append(conns, c)
			})
		}
	}
}
