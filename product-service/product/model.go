package product

import "time"

type ProductModel struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name"`
	Href        string     `json:"href,omitempty"`
	Price       string     `json:"price"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"createdAt,omitzero"`
	UpdatedAt   time.Time  `json:"updatedAt,omitzero"`
	DeletedAt   *time.Time `json:"deletedAt,omitzero"`
}
