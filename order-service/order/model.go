package order

import "time"

type Item struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"` // Price per unit
}

type Order struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customer_id"`
	Items      []Item  `json:"items"`
	TotalPrice float64 `json:"total_price"`
	Status     string  `json:"status"`     // e.g., "pending", "completed", "canceled"
	CreatedAt  string  `json:"created_at"` // ISO 8601 format
	UpdatedAt  string  `json:"updated_at"` // ISO 8601 format
}

type UserModel struct {
	ID        string `json:"id" bson:"_id"`
	Href      string `json:"href,omitempty" bson:"-"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Email     string `json:"email" bson:"email,unique"`
	Avatar    string `json:"avatar,omitempty"`
	Password  string `json:"password,omitempty" bson:"-"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProductModel struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name"`
	Href        string    `json:"href,omitempty"`
	Price       string    `json:"price"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitzero"`
	UpdatedAt   time.Time `json:"updatedAt,omitzero"`
}
