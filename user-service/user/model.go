package user

import "time"

type UserModel struct {
	ID        string     `json:"id" bson:"_id"`
	Href      string     `json:"href,omitempty" bson:"-"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Username  string     `json:"username"`
	Email     string     `json:"email" bson:"email,unique"`
	Avatar    string     `json:"avatar,omitempty"`
	Password  string     `json:"password,omitempty" bson:"-"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	DeletedAt *time.Time `json:"-" bson:"deleted_at"`
}
