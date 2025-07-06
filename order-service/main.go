package main

import (
	"context"
	"fmt"
	"os"
	"time"

	config "github.com/sing3demons/go-common-kp/kp/configs"
	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"github.com/sing3demons/go-common-kp/kp/pkg/logger"
	"github.com/sing3demons/go-order-service/order"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongo(conf *config.Config) *mongo.Database {
	uri := conf.Get("MONGO_URI")

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	err = mongoClient.Ping(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	db := mongoClient.Database("order_service") // Replace with your database name
	fmt.Println("Connected to MongoDB!")
	return db
}

func main() {
	conf := config.NewConfig()
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs"
	}
	conf.LoadEnv(path)
	logApp := logger.NewLogger(conf.Log.App)
	defer logApp.Sync()

	logDetail := logger.NewLogger(conf.Log.Detail)
	defer logDetail.Sync()
	logSummary := logger.NewLogger(conf.Log.Summary)
	defer logSummary.Sync()

	mongoDB := ConnectMongo(conf)
	app := kp.NewApplication(conf, logApp)
	app.LogDetail(logDetail)
	app.LogSummary(logSummary)
	app.StartKafka()
	app.CreateTopic("create_order_history")

	app.Get("/healthz", func(ctx *kp.Context) error {
		return ctx.JSON(200, "OK")
	})
	app.Consumer("create_order_history", func(ctx *kp.Context) error {
		// data := map[string]any{
		// 	"body": map[string]any{
		// 		"order_id":    o.ID,
		// 		"customer":    user,
		// 		"products":    products,
		// 		"total_price": o.TotalPrice,
		// 	},
		// }

		var data map[string]any
		if err := ctx.Bind(&data); err != nil {
			return ctx.JSON(400, "Invalid request")
		}
		summary := logger.LogEventTag{
			Node:        "mongo",
			Command:     "insert_order_history",
			Code:        "200",
			Description: "success",
		}

		ctx.Log().Info(logger.NewDBRequest(logger.INSERT, "insert order history"), map[string]any{
			"collection": "order_history",
			"data":       data["body"],
		})
		start := time.Now()

		r, err := mongoDB.Collection("order_history").InsertOne(ctx, data["body"])
		summary.ResTime = time.Since(start).Milliseconds()
		if err != nil {
			summary.Code = "500"
			summary.Description = "failed to insert order history"

			ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.INSERT, "insert order history failed"), map[string]string{
				"error": err.Error(),
			})
			return ctx.JSON(500, "Failed to insert order history")
		}

		ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.INSERT, "insert order history success"), map[string]any{
			"collection": "order_history",
			"result":     r,
		})
		return ctx.JSON(200, "Consumer is running")
	})

	order.RegisterRoutes(app, mongoDB.Collection("orders"))
	app.Start()
}
