package user

import "github.com/sing3demons/go-common-kp/kp/pkg/kp"

 
type Service interface {
	CreateUser(ctx *kp.Context, user *UserModel) error
	GetUserByID(ctx *kp.Context, id string) (*UserModel, error)
	GetAllUsers(ctx *kp.Context) ([]*UserModel, error)
	DeleteUser(ctx *kp.Context, id string) error
}

type userService struct {
	repo Repository
}

func NewUserService(repo Repository) Service {
	return &userService{
		repo: repo,
	}
}

func (s *userService) CreateUser(ctx *kp.Context, user *UserModel) error {
	return s.repo.CreateUser(ctx, user)
}

func (s *userService) GetUserByID(ctx *kp.Context, id string) (*UserModel, error) {
	return s.repo.GetUserByID(ctx, id)
}
func (s *userService) GetAllUsers(ctx *kp.Context) ([]*UserModel, error) {
	return s.repo.GetAllUsers(ctx)
}

func (s *userService) DeleteUser(ctx *kp.Context, id string) error {
	return s.repo.DeleteUser(ctx, id)
}