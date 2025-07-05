package order

import (
	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterRoutes(app kp.IApplication, col *mongo.Collection) {
	repo := NewRepository(col)
	service := NewOrderService(repo)
	handler := NewHandler(service)
	app.Post("/orders", handler.HandleCreateOrder)
}
