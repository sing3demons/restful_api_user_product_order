package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
	config "github.com/sing3demons/go-common-kp/kp/configs"
	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"github.com/sing3demons/go-product-service/product"
)

func NewDB(conf *config.Config) (*sql.DB, error) {
	databaseSource := conf.Get("DB_HOST")

	db, err := sql.Open("postgres", databaseSource)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	conf := config.NewConfig()
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs"
	}
	conf.LoadEnv(path)

	db, err := NewDB(conf)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer db.Close()

	app := kp.NewApplication(conf)
	// app.StartKafka()

	app.Get("/healthz", func(ctx *kp.Context) error {
		if err := db.Ping(); err != nil {
			log.Printf("Database connection error: %v", err)
			return ctx.JSON(500, "Down")
		}
		ctx.Debug("Database connection is healthy")
		return ctx.JSON(200, "UP")
	})

	// Register product routes
	product.RegisterRoutes(app, db)

	app.Start()
}
