package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	mysqlUser := getEnv("MYSQL_USER", "root")
	mysqlPassword := getEnv("MYSQL_PASSWORD", "my-secret-pw")
	mysqlHost := getEnv("MYSQL_HOST", "localhost:3306")
	mysqlDatabase := getEnv("MYSQL_DB", "gigmile")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		mysqlUser, mysqlPassword, mysqlHost, mysqlDatabase)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping MySQL: %v\nDSN: %s:%s@tcp(%s)/%s",
			err, mysqlUser, "***", mysqlHost, mysqlDatabase)
	}

	fmt.Println("Connected to MySQL successfully")

	// Seed customers
	customers := []struct {
		id         string
		assetValue int64
		termWeeks  int
	}{
		{"GIG00001", 100000000, 50}, // N1,000,000 in kobo
		{"GIG00002", 100000000, 50},
		{"GIG00003", 100000000, 50},
		{"GIG00004", 100000000, 50},
		{"GIG00005", 100000000, 50},
	}

	deploymentDate := time.Now().AddDate(0, 0, -14)
	query := `
		INSERT INTO customers (id, asset_value, outstanding_balance, total_paid,
		                       repayment_term_weeks, deployment_date, status, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		    asset_value = VALUES(asset_value),
		    outstanding_balance = VALUES(outstanding_balance),
		    total_paid = VALUES(total_paid)
	`

	for _, c := range customers {
		_, err := db.Exec(query,
			c.id,
			c.assetValue,
			c.assetValue, // outstanding_balance starts at full asset value
			0,            // total_paid starts at 0
			c.termWeeks,
			deploymentDate,
			"ACTIVE",
			1, // version
		)

		if err != nil {
			log.Fatalf("Failed to seed customer %s: %v", c.id, err)
		}

		fmt.Printf("Seeded customer: %s (Asset: N%d, Term: %d weeks)\n",
			c.id, c.assetValue/100, c.termWeeks)
	}

	fmt.Println("\nSeed completed successfully!")
	fmt.Println("You can now test the API with customer IDs: GIG00001 to GIG00005")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
