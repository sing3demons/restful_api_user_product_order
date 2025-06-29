package product

import (
	"github.com/sing3demons/go-product-service/pkg/kp"
	"github.com/sing3demons/go-product-service/pkg/logger"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CreateProduct handles the creation of a new product
func (h *Handler) CreateProduct(ctx *kp.Context) error {
	var product ProductModel

	summary := logger.LogEventTag{
		Node:        "client",
		Command:     "create_user",
		Code:        "200",
		Description: "",
	}
	if err := ctx.Bind(&product); err != nil {
		summary.Code = "400"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewInbound("create product error", ""), map[string]any{
			"error": err.Error(),
		})
		return ctx.JSON(400, map[string]string{
			"error": "invalid_request",
		})
	}

	if product.Name == "" {
		summary.Code = "400"
		summary.Description = "invalid_request"
		ctx.Log().SetSummary(summary).Error(logger.NewInbound("create product error", ""), map[string]any{
			"error": "name and price are required",
		})
		return ctx.JSON(400, map[string]string{
			"error": "name and price are required",
		})
	}
	if err := h.service.CreateProduct(ctx, &product); err != nil {
		return ctx.JSON(500, map[string]string{
			"error": "internal_server_error",
		})
	}

	return ctx.JSON(201, map[string]any{
		"message": "success",
		"product": product,
	})
}

// GetProductByID handles fetching a product by its ID
func (h *Handler) GetProductByID(ctx *kp.Context) error {
	id := ctx.PathParam("id")
	if id == "" {
		return ctx.JSON(400, map[string]string{
			"error": "invalid_request",
		})
	}

	product, err := h.service.GetProductByID(ctx, id)
	if err != nil {
		ctx.Log().Error(logger.NewInbound("get product error", ""), map[string]any{
			"error": err.Error(),
		})
		return ctx.JSON(500, map[string]string{
			"error": "internal_server_error",
		})
	}

	if product == nil {
		return ctx.JSON(404, map[string]string{
			"error": "product_not_found",
		})
	}

	return ctx.JSON(200, product)
}

// FindProducts handles fetching all products with optional filtering
func (h *Handler) FindProducts(ctx *kp.Context) error {
	summary := logger.LogEventTag{
		Node:        "client",
		Command:     "find_products",
		Code:        "200",
		Description: "",
	}

	ctx.Log().SetSummary(summary).Info(logger.NewInbound("find products", ""), map[string]any{
		"name":  ctx.Param("name"),
		"limit": ctx.Param("limit"),
	})

	products, err := h.service.FindProducts(ctx)
	if err != nil {
		return ctx.JSON(500, map[string]string{
			"error": "internal_server_error",
		})
	}

	return ctx.JSON(200, map[string]any{
		"products": products,
	})
}

// DeleteProduct handles the deletion of a product by its ID
func (h *Handler) DeleteProduct(ctx *kp.Context) error {
	summary := logger.LogEventTag{
		Node:        "client",
		Command:     "delete_product",
		Code:        "200",
		Description: "",
	}
	id := ctx.PathParam("id")
	if id == "" {
		summary.Code = "400"
		summary.Description = "invalid_request"
		ctx.Log().SetSummary(summary).Error(logger.NewInbound("delete product error", ""), map[string]any{
			"error": "product ID is required",
		})
		return ctx.JSON(400, map[string]string{
			"error": "invalid_request",
		})
	}
	ctx.Log().SetSummary(summary).Info(logger.NewInbound("delete product", ""), map[string]any{
		"param": map[string]string{
			"key":   "id",
			"value": id,
		},
	})

	if err := h.service.DeleteProduct(ctx, id); err != nil {
		return ctx.JSON(500, map[string]string{
			"error": "internal_server_error",
		})
	}

	return ctx.JSON(204, nil)
}
