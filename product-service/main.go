package main

import (
	"database/sql"
	"fmt"

	config "github.com/sing3demons/go-product-service/configs"
	"github.com/sing3demons/go-product-service/pkg/kp"
	"github.com/sing3demons/go-product-service/pkg/logger"
	"github.com/sing3demons/go-product-service/product"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func NewDB() (*sql.DB, error) {
	databaseSource := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable", "localhost", 5432, "root", "password", "product_master")

	db, err := sql.Open("postgres", databaseSource)
	if err != nil {
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
	conf.LoadEnv("configs")

	logApp := logger.NewLogger(conf.Log.App)
	defer logApp.Sync()

	logDetail := logger.NewLogger(conf.Log.Detail)
	defer logDetail.Sync()
	logSummary := logger.NewLogger(conf.Log.Summary)
	defer logSummary.Sync()

	db, err := NewDB()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer db.Close()

	app := kp.NewApplication(conf, logApp)
	app.LogDetail(logDetail)
	app.LogSummary(logSummary)
	// app.StartKafka()

	app.Get("/healthz", func(ctx *kp.Context) error {
		return ctx.JSON(200, "OK")
	})

	// Register product routes
	product.RegisterRoutes(app, db)

	app.Start()
}
