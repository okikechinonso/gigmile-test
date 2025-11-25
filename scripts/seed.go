package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/go-redis/redis/v8"
)

func main() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	fmt.Println("Connected to Redis successfully")

	// Seed customers
	customers := []struct {
		id         string
		assetValue int64
		termWeeks  int
	}{
		{"GIG00001", 100000000, 50}, 
		{"GIG00002", 100000000, 50},
		{"GIG00003", 100000000, 50},
		{"GIG00004", 100000000, 50},
		{"GIG00005", 100000000, 50},
	}

	deploymentDate := time.Now().AddDate(0, 0, -14) 

	for _, c := range customers {
		customer, err := domain.NewCustomer(c.id, c.assetValue, c.termWeeks, deploymentDate)
		if err != nil {
			log.Fatalf("Failed to create customer %s: %v", c.id, err)
		}

		data, err := json.Marshal(customer)
		if err != nil {
			log.Fatalf("Failed to marshal customer %s: %v", c.id, err)
		}

		key := fmt.Sprintf("customer:%s", c.id)
		if err := client.Set(ctx, key, data, 0).Err(); err != nil {
			log.Fatalf("Failed to save customer %s: %v", c.id, err)
		}

		fmt.Printf("Seeded customer: %s (Asset: N%d, Term: %d weeks)\n",
			c.id, c.assetValue/100, c.termWeeks)
	}

	fmt.Println("\nSeed completed successfully!")
	fmt.Println("You can now test the API with customer IDs: GIG00001 to GIG00005")
}
