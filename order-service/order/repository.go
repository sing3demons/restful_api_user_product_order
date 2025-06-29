package order

import (
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/go-order-service/pkg/kp"
	"github.com/sing3demons/go-order-service/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository interface {
	CreateOrder(ctx *kp.Context, order Order) (Order, error)
}

type repository struct {
	col *mongo.Collection
}

func NewRepository(col *mongo.Collection) Repository {
	return &repository{
		col: col,
	}
}

func (r *repository) CreateOrder(ctx *kp.Context, order Order) (Order, error) {
	start := time.Now()
	summary := logger.LogEventTag{
		Node:        "mongo",
		Command:     "create_order",
		Code:        "200",
		Description: "success",
	}

	ctx.Log().Info(logger.NewDBRequest(logger.INSERT, "insert order"), map[string]any{
		"Order": order,
	})

	id, err := uuid.NewV7()
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to generate order ID"
		ctx.Log().Error(logger.NewDBRequest(logger.INSERT, "insert order"), map[string]string{
			"error": err.Error(),
		})
		return Order{}, err
	}
	order.ID = id.String()

	result, err := r.col.InsertOne(ctx, order)
	summary.ResTime = time.Since(start).Milliseconds()
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to insert order"
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.INSERT, "insert order failed"), map[string]string{
			"error": err.Error(),
		})
		return Order{}, err
	}

	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.INSERT, "insert order success"), map[string]any{
		"Return": result,
	})

	return order, nil
}
