package product

import "github.com/sing3demons/go-common-kp/kp/pkg/kp"

type Service interface {
	GetProductByID(ctx *kp.Context, id string) (*ProductModel, error)
	CreateProduct(ctx *kp.Context, product *ProductModel) error
	FindProducts(ctx *kp.Context) ([]*ProductModel, error)
	DeleteProduct(ctx *kp.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}
func (s *service) CreateProduct(ctx *kp.Context, product *ProductModel) error {
	return s.repo.CreateProduct(ctx, product)
}

func (s *service) GetProductByID(ctx *kp.Context, id string) (*ProductModel, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) FindProducts(ctx *kp.Context) ([]*ProductModel, error) {
	return s.repo.FindProducts(ctx)
}

func (s *service) DeleteProduct(ctx *kp.Context, id string) error {
	return s.repo.DeleteProduct(ctx, id)
}
