package product

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/sing3demons/go-product-service/pkg/kp"
	"github.com/sing3demons/go-product-service/pkg/logger"
)

type Repository interface {
	FindByID(ctx *kp.Context, id string) (*ProductModel, error)
	CreateProduct(ctx *kp.Context, product *ProductModel) error
	FindProducts(ctx *kp.Context) ([]*ProductModel, error)
	DeleteProduct(ctx *kp.Context, id string) error
}
type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindByID(ctx *kp.Context, id string) (*ProductModel, error) {
	query := `SELECT id, name, price, description, created_at, updated_at FROM products WHERE id = $1 AND deleted_at IS NULL`
	row := r.db.QueryRow(query, id)

	var product ProductModel
	err := row.Scan(&product.ID, &product.Name, &product.Price, &product.Description, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No product found
		}
		return nil, err // Other error
	}
	product.Href = "/products/" + product.ID

	return &product, nil
}

func (r *repository) CreateProduct(ctx *kp.Context, product *ProductModel) error {
	start := time.Now()
	summary := logger.EventTag("progress", "insert_product", "200", "success")

	query := `INSERT INTO products (name, price, description, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`

	ctx.Log().Info(logger.NewDBRequest(logger.INSERT, "create product"), map[string]any{
		"query":  query,
		"params": []any{product.Name, product.Price, product.Description},
	})
	var id string
	err := r.db.QueryRow(query, product.Name, product.Price, product.Description).Scan(&id)

	summary.ResTime = time.Since(start).Milliseconds()
	if err != nil {
		summary.Code = "500"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.INSERT, "create product error"), map[string]any{
			"error": err.Error(),
		})
		return err
	}
	product.ID = id

	summary.Code = "201"
	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.INSERT, "create product success"), map[string]any{
		"Return": product,
	})
	return nil
}

func (r *repository) FindProducts(ctx *kp.Context) ([]*ProductModel, error) {
	start := time.Now()
	summary := logger.EventTag("progress", "find_products", "200", "success")
	baseQuery := `
	SELECT id, name, price, description, created_at, updated_at
	FROM products
	WHERE deleted_at IS NULL`

	name := ctx.Param("name")
	var args []any
	argIndex := 1

	if name != "" {
		if name != "" {
			baseQuery += fmt.Sprintf(" AND name ILIKE '%%' || $%d || '%%'", argIndex)
			args = append(args, name)
			argIndex++
		}

	}

	limit := 0
	l := ctx.Param("limit")
	if l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if limit == 0 {
		limit = 20 // Default limit if not specified
	}
	baseQuery += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argIndex)
	args = append(args, limit)

	ctx.Log().Info(logger.NewDBRequest(logger.QUERY, "find products"), map[string]any{
		"query":  baseQuery,
		"params": args,
	})

	rows, err := r.db.Query(baseQuery, args...)
	summary.ResTime = time.Since(start).Milliseconds()
	if err != nil {
		summary.Code = "500"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.QUERY, "find products error"), map[string]any{
			"error": err.Error(),
		})
		return nil, err
	}
	defer rows.Close()

	var products []*ProductModel
	for rows.Next() {
		var product ProductModel
		err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Description, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			summary.Code = "500"
			summary.Description = err.Error()
			ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.QUERY, "scan product error"), map[string]any{
				"error": err.Error(),
			})
			return nil, err
		}
		product.Href = "/products/" + product.ID
		products = append(products, &product)
	}

	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.QUERY, "find products success"), map[string]any{
		"Return": products,
	})

	return products, nil
}

// DeleteProduct is not implemented in the repository interface, but you can add it if needed.
func (r *repository) DeleteProduct(ctx *kp.Context, id string) error {
	start := time.Now()
	summary := logger.EventTag("progress", "delete_product", "200", "success")
	query := `UPDATE products SET deleted_at = NOW() WHERE id = $1`
	ctx.Log().Info(logger.NewDBRequest(logger.UPDATE, "delete product"), map[string]any{
		"query":  query,
		"params": []any{id},
	})

	result, err := r.db.Exec(query, id)
	summary.ResTime = time.Since(start).Milliseconds()
	if err != nil {
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.UPDATE, "delete product error"), map[string]any{
			"error": err.Error(),
		})
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		summary.Code = "404"
		summary.Description = "product not found"
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.UPDATE, "delete product not found"), map[string]any{
			"error": fmt.Sprintf("product with id %s not found", id),
		})
		return fmt.Errorf("product with id %s not found", id)
	}

	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.UPDATE, "delete product success"), map[string]any{
		"rows_affected": rowsAffected,
	})
	return nil
}
