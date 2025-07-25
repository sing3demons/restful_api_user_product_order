package user

import (
	"context"

	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RegisterRoutes(app kp.IApplication, col *mongo.Collection) {
	indexEmailModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
		},
		Options: options.Index().
			SetName("unique_email_if_not_deleted").
			SetUnique(true).
			SetPartialFilterExpression(bson.D{
				{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: nil}}},
			}),
	}

	indexUsernameModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "username", Value: 1},
		},
		Options: options.Index().
			SetName("unique_username_if_not_deleted").
			SetUnique(true).
			SetPartialFilterExpression(bson.D{
				{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: nil}}},
			}),
	}

	col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{indexEmailModel, indexUsernameModel})
	repo := NewUserRepository(col)
	svc := NewUserService(repo)
	handler := NewHandler(svc)

	// User routes
	app.Post("/users", handler.CreateUser)
	// username or email
	app.Get("/users/{key}/{value}", handler.GetUser)
	app.Get("/users/{id}", handler.GetUserByID)
	app.Get("/users", handler.GetAllUsers)
	app.Delete("/users/{id}", handler.DeleteUser)

}
