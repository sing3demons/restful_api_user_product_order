package main

import (
	"context"
	"fmt"
	"os"

	config "github.com/sing3demons/go-common-kp/kp/configs"
	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"github.com/sing3demons/go-common-kp/kp/pkg/logger"

	"github.com/sing3demons/go-user-service/user"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongo(conf *config.Config) *mongo.Database {
	uri := conf.Get("MONGO_URI")
	fmt.Println("Connecting to MongoDB at:", uri)

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	err = mongoClient.Ping(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	db := mongoClient.Database("user_service")
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
	// app.StartKafka()

	app.Get("/healthz", func(ctx *kp.Context) error {
		return ctx.JSON(200, "OK")
	})

	user.RegisterRoutes(app, mongoDB.Collection("users"))
	app.Start()
}
