package order

import (
	"errors"

	"github.com/sing3demons/go-order-service/pkg/kp"
	"github.com/sing3demons/go-order-service/pkg/logger"
)

type Handler struct {
	service OrderService
}

func NewHandler(service OrderService) *Handler {
	return &Handler{
		service: service,
	}
}

// HandleCreateOrder handles the creation of a new order
func (h *Handler) HandleCreateOrder(ctx *kp.Context) error {
	summary := logger.LogEventTag{
		Node:        "client",
		Command:     "create_order",
		Code:        "200",
		Description: "",
	}
	var req Order
	if err := ctx.Bind(&req); err != nil {
		summary.Code = "400"
		summary.Description = "invalid_request"
		ctx.Log().SetSummary(summary).Error(logger.NewInbound("create order failed", ""), map[string]string{
			"error": err.Error(),
		})
		return ctx.JSON(400, map[string]string{
			"error": "invalid request",
		})
	}

	if err := h.validateOrder(ctx, req); err != nil {
		return ctx.JSON(400, map[string]string{
			"error": err.Error(),
		})
	}
	ctx.Log().SetSummary(summary).Info(logger.NewInbound("create order", ""), map[string]any{
		"body": req,
	})

	order, err := h.service.CreateOrder(ctx, req)
	if err != nil {
		return ctx.JSON(500, map[string]string{
			"error": "failed to create order",
		})
	}

	return ctx.JSON(200, map[string]any{
		"order_id":    order.ID,
		"customer_id": order.CustomerID,
		"items":       order.Items,
		"total_price": order.TotalPrice,
		"status":      order.Status,
		"created_at":  order.CreatedAt,
		"updated_at":  order.UpdatedAt,
	})
}

func (h *Handler) validateOrder(ctx *kp.Context, req Order) error {
	summary := logger.LogEventTag{
		Node:        "client",
		Command:     "create_order",
		Code:        "400",
		Description: "invalid_request",
	}

	desc := "create order request validation failed"

	// validate the request
	if req.CustomerID == "" {
		ctx.Log().SetSummary(summary).Error(logger.NewInbound(desc, ""), map[string]string{
			"error": "customer_id is required",
		})
		return errors.New(summary.Description)
	}

	if len(req.Items) == 0 {
		ctx.Log().SetSummary(summary).Error(logger.NewInbound(desc, ""), map[string]string{
			"error": "items are required",
		})
		return errors.New(summary.Description)

	}

	if len(req.Items) > 0 {
		for _, item := range req.Items {
			if item.ID == "" || item.Name == "" || item.Quantity <= 0 || item.Price <= 0 {
				ctx.Log().SetSummary(summary).Error(logger.NewInbound(desc, ""), map[string]string{
					"error": "invalid item data",
				})
				return errors.New(summary.Description)

			}
		}
	}

	if req.TotalPrice <= 0 {
		ctx.Log().SetSummary(summary).Error(logger.NewInbound("create order", ""), map[string]string{
			"error": "total_price must be greater than 0",
		})
		return errors.New(summary.Description)
	}
	return nil
}
