package order

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sing3demons/go-order-service/pkg/kp"
	"github.com/sing3demons/go-order-service/pkg/logger"
)

type OrderService interface {
	CreateOrder(ctx *kp.Context, order Order) (Order, error)
	// GetOrderByID(id string) (Order, error)
	// UpdateOrder(order Order) (Order, error)
	// DeleteOrder(id string) error
	// ListOrders(customerID string) ([]Order, error)
	// CalculateTotalPrice(order Order) float64
}
type orderService struct {
	repo Repository
}

func NewOrderService(repo Repository) OrderService {
	return &orderService{
		repo: repo,
	}
}

func (s *orderService) CreateOrder(ctx *kp.Context, order Order) (Order, error) {

	user, err := getUserByID(ctx, order.CustomerID)
	if err != nil {
		return Order{}, err
	}

	products := []ProductModel{}
	for _, item := range order.Items {
		product, err := getProductByID(ctx, item.ID)
		if err != nil {
			return Order{}, err
		}
		products = append(products, product)
	}

	o, err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		return Order{}, err
	}

	data := map[string]any{
		"body": map[string]any{
			"order_id":    o.ID,
			"customer":    user,
			"products":    products,
			"total_price": o.TotalPrice,
		},
	}

	start := time.Now()
	summary := logger.LogEventTag{
		Node:        "kafka",
		Command:     "create_order_history",
		Code:        "200",
		Description: "success",
		ResTime:     0,
	}
	message, err := json.Marshal(data)
	if err != nil {
		return Order{}, err
	}
	ctx.Log().Info(logger.NewProducing(summary.Command, ""), map[string]any{
		"topic":  summary.Command,
		"value":  string(message),
		"broker": "localhost:9092",
	})
	if err := ctx.Publish(ctx, summary.Command, message); err != nil {
		summary.Code = "500"
		summary.Description = "failed to publish order history"
		summary.ResTime = time.Since(start).Milliseconds()

		ctx.Log().SetSummary(summary).Error(logger.NewProduced(summary.Command, ""), map[string]string{
			"error": err.Error(),
		})
		return Order{}, err
	}
	summary.ResTime = time.Since(start).Milliseconds()
	ctx.Log().SetSummary(summary).Info(logger.NewProduced(summary.Command, ""), map[string]any{
		"topic":  summary.Command,
		"broker": "localhost:9092",
	})
	return o, nil
}

type HttpRequest struct {
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Params   map[string]string `json:"params"`
	Protocol string            `json:"protocol"`
	Method   string            `json:"method"`
	Timeout  time.Duration     `json:"timeout"`
}

const contentTypeHeader = "Content-Type"

func getUserByID(ctx *kp.Context, userID string) (UserModel, error) {
	start := time.Now()
	summary := logger.LogEventTag{
		Node:        "user_service",
		Command:     "get_user_by_id",
		Code:        "200",
		Description: "success",
	}

	// userServiceURL := ctx.GetConfig().GetOrDefault("USER_SERVICE_URL", "http://localhost:8080")
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8080" // Default URL if not set
	}

	httpRequest := HttpRequest{
		URL:      userServiceURL + "/users/" + userID,
		Headers:  map[string]string{contentTypeHeader: "application/json"},
		Params:   map[string]string{"user_id": userID},
		Protocol: "http",
		Method:   http.MethodGet,
		Timeout:  10 * time.Second,
	}

	ctx.Log().SetSummary(summary).Info(logger.NewHTTPRequest("get user by ID", ""), map[string]any{
		"uri":      httpRequest.URL,
		"headers":  httpRequest.Headers,
		"params":   httpRequest.Params,
		"protocol": httpRequest.Protocol,
		"method":   httpRequest.Method,
		"timeout":  httpRequest.Timeout,
	})
	req, err := http.NewRequest(http.MethodGet, httpRequest.URL, nil)
	if err != nil {
		return UserModel{}, err
	}
	req.Header.Set(contentTypeHeader, httpRequest.Headers[contentTypeHeader])

	httpClient := &http.Client{
		Timeout: httpRequest.Timeout,
	}

	resp, err := httpClient.Do(req)
	summary.ResTime = time.Since(start).Milliseconds()
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to get user by ID"
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("http get user", ""), map[string]string{
			"error": err.Error(),
		})
		return UserModel{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		summary.Code = fmt.Sprintf("%d", resp.StatusCode)
		summary.Description = resp.Status
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("get user by id failed", ""), map[string]string{
			"error": fmt.Sprintf("failed to get user by ID: %s", resp.Status),
		})
		return UserModel{}, fmt.Errorf("failed to get user by ID: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to read response body"
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("get user by ID failed", ""), map[string]string{
			"error": err.Error(),
		})
		return UserModel{}, err
	}

	var user UserModel
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		summary.Code = "500"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("get user by ID failed", ""), map[string]string{
			"error": err.Error(),
		})
		return UserModel{}, err
	}

	ctx.Log().SetSummary(summary).Info(logger.NewHTTPResponse("get user by ID success", ""), map[string]any{
		"Headers": resp.Header,
		"Status":  resp.Status,
		"Body":    user,
	})
	return user, nil
}

func getProductByID(ctx *kp.Context, productID string) (ProductModel, error) {
	start := time.Now()
	summary := logger.LogEventTag{
		Node:        "product_service",
		Command:     "get_product_by_id",
		Code:        "200",
		Description: "success",
	}

	// productServiceURL := ctx.GetConfig().GetOrDefault("PRODUCT_SERVICE_URL", "http://localhost:8082")
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "http://localhost:8082" // Default URL if not set
	}
	httpRequest := HttpRequest{
		URL:      productServiceURL + "/products/" + productID,
		Headers:  map[string]string{contentTypeHeader: "application/json"},
		Params:   map[string]string{"product_id": productID},
		Protocol: "http",
		Method:   http.MethodGet,
		Timeout:  10 * time.Second,
	}

	ctx.Log().Info(logger.NewHTTPRequest("get product by ID", ""), map[string]any{
		"uri":      httpRequest.URL,
		"headers":  httpRequest.Headers,
		"params":   httpRequest.Params,
		"protocol": httpRequest.Protocol,
		"method":   httpRequest.Method,
		"timeout":  httpRequest.Timeout,
	})
	req, err := http.NewRequest(http.MethodGet, httpRequest.URL, nil)
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to create HTTP request"
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("http get product", ""), map[string]string{
			"error": err.Error(),
		})
		return ProductModel{}, err
	}
	req.Header.Set(contentTypeHeader, httpRequest.Headers[contentTypeHeader])

	httpClient := &http.Client{
		Timeout: httpRequest.Timeout,
	}

	resp, err := httpClient.Do(req)
	summary.ResTime = time.Since(start).Milliseconds()
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to get product by ID"
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("http get product", ""), map[string]string{
			"error": err.Error(),
		})
		return ProductModel{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		summary.Code = fmt.Sprintf("%d", resp.StatusCode)
		summary.Description = resp.Status
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("get product by ID failed", ""), map[string]string{
			"error": fmt.Sprintf("failed to get product by ID: %s", resp.Status),
		})
		return ProductModel{}, fmt.Errorf("failed to get product by ID: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		summary.Code = "500"
		summary.Description = "failed to read response body"
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("get product by ID failed", ""), map[string]string{
			"error": err.Error(),
		})
		return ProductModel{}, err
	}
	var product ProductModel
	if err := json.Unmarshal(bodyBytes, &product); err != nil {
		summary.Code = "500"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewHTTPResponse("get product by ID failed", ""), map[string]string{
			"error": err.Error(),
		})
		return ProductModel{}, err
	}
	ctx.Log().SetSummary(summary).Info(logger.NewHTTPResponse("get product by ID success", ""), map[string]any{
		"Headers": resp.Header,
		"Status":  resp.Status,
		"Body":    product,
	})
	return product, nil
}
