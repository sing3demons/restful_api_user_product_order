package product

import (
	"database/sql"
	"fmt"

	"github.com/sing3demons/go-product-service/pkg/kp"
)

const createTable = `
CREATE TABLE IF NOT EXISTS products (
   id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    price       NUMERIC NOT NULL,
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

    DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_indexes
            WHERE indexname = 'unique_name_if_not_deleted'
        ) THEN
            CREATE UNIQUE INDEX unique_name_if_not_deleted
            ON products(name)
            WHERE deleted_at IS NULL;
        END IF;
    END
    $$;`

func RegisterRoutes(app kp.IApplication, db *sql.DB) {
	_, err := db.Exec(createTable)
	if err != nil {
		fmt.Println("Error creating products table:", err)
		return
	}

	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	app.Post("/products", handler.CreateProduct)
	app.Get("/products/{id}", handler.GetProductByID)
	app.Get("/products", handler.FindProducts)
	app.Delete("/products/{id}", handler.DeleteProduct)
}
