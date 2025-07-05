package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"github.com/sing3demons/go-common-kp/kp/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository interface {
	GetUserByID(ctx *kp.Context, id string) (*UserModel, error)
	GetAllUsers(ctx *kp.Context) ([]*UserModel, error)
	CreateUser(ctx *kp.Context, user *UserModel) error
	// UpdateUser(ctx *kp.Context, user *UserModel) error
	DeleteUser(ctx *kp.Context, id string) error
}

type userRepository struct {
	col *mongo.Collection
}

func NewUserRepository(col *mongo.Collection) Repository {
	return &userRepository{
		col: col,
	}
}

type ProcessMongoReq struct {
	Collection string `json:"collection"`
	Method     string `json:"method"`
	Query      any    `json:"query"`
	Document   any    `json:"document"`
	Options    any    `json:"options"`
}

func formatMapAsMongoShell(m map[string]interface{}) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf(`%s:%s`, k, formatMongoShellValue(v)))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatMongoShellValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, val)
	case int, int32, int64, float32, float64, bool:
		return fmt.Sprintf(`%v`, val)
	case nil:
		return "null"
	case primitive.Null:
		return "null"
	case map[string]interface{}:
		return formatMapAsMongoShell(val)
	case []interface{}:
		var parts []string
		for _, item := range val {
			parts = append(parts, formatMongoShellValue(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		// fallback: use reflect to catch primitive values
		if reflect.TypeOf(val).Kind() == reflect.Struct && reflect.DeepEqual(val, primitive.Null{}) {
			return "null"
		}
		return fmt.Sprintf(`"%v"`, val)
	}
}

func (r *ProcessMongoReq) RawString() string {
	method := strings.ToLower(r.Method)

	if strings.HasPrefix(method, "insert") {
		jsonDocumentBytes, _ := json.Marshal(r.Document)
		return fmt.Sprintf("db.%s.%s(%s)", r.Collection, r.Method, string(jsonDocumentBytes))
	} else if strings.HasPrefix(method, "update") {
		queryMap, ok := r.Query.(map[string]any)
		if !ok || len(queryMap) == 0 {
			queryMap = map[string]any{}
		}
		queryStr := formatMapAsMongoShell(queryMap)

		updateStr := "{}"
		if updateMap, ok := r.Document.(map[string]any); ok && len(updateMap) > 0 {
			updateStr = formatMapAsMongoShell(updateMap)
		}

		return fmt.Sprintf("db.%s.%s(%s, %s)", r.Collection, r.Method, queryStr, updateStr)
	}

	if queryMap, ok := r.Query.(map[string]any); ok {
		queryStr := formatMapAsMongoShell(queryMap)
		return fmt.Sprintf("db.%s.%s(%s)", r.Collection, r.Method, queryStr)
	}

	return fmt.Sprintf("db.%s.%s(%v)", r.Collection, r.Method, r.Query)
}

func (r *userRepository) CreateUser(ctx *kp.Context, user *UserModel) error {
	desc := "insert user"
	cmd := "insert_user"
	node := "mongo"

	start := time.Now()
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	user.ID = id.String()
	user.CreatedAt = start.Format(time.RFC3339)
	user.UpdatedAt = start.Format(time.RFC3339)
	user.DeletedAt = nil

	processReqLog := ProcessMongoReq{
		Collection: r.col.Name(),
		Method:     "InsertOne",
		Query:      nil,
		Document:   user,
		Options:    nil,
	}

	maskingOption := []logger.MaskingOptionDto{
		{
			MaskingField: "Body.document.email",
			MaskingType:  logger.Email,
		},
		{
			MaskingField: "Body.document.first_name",
			MaskingType:  logger.Firstname,
		},
		{
			MaskingField: "Body.document.last_name",
			MaskingType:  logger.Lastname,
		},
	}

	ctx.Log().Info(logger.NewDBRequest(logger.INSERT, desc), map[string]any{
		"Body": processReqLog,
		"Raw":  processReqLog.RawString(),
	}, maskingOption...)

	result, err := r.col.InsertOne(ctx, user)
	end := time.Since(start)
	if err != nil {
		code := "500"
		desc := err.Error()
		if mongo.IsDuplicateKeyError(err) {
			code = "409"
			desc = "duplicate_key"
		}
		ctx.Log().SetSummary(logger.LogEventTag{
			Node:        node,
			Command:     cmd,
			Code:        code,
			Description: desc,
			ResTime:     end.Microseconds(),
		}).Error(logger.NewDBResponse(logger.INSERT, desc), map[string]any{
			"Error": err.Error(),
			"Raw":   processReqLog.RawString(),
		})
		return err
	}

	ctx.Log().SetSummary(logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "success",
		ResTime:     end.Microseconds(),
	}).Info(logger.NewDBResponse(logger.INSERT, desc), map[string]any{
		"Return": result,
	})

	return nil
}

func (r *userRepository) GetUserByID(ctx *kp.Context, id string) (*UserModel, error) {
	desc := "get user by id"
	cmd := "get_user_by_id"
	node := "mongo"

	start := time.Now()

	filter := map[string]any{
		"deleted_at": primitive.Null{},
		"_id":        id,
	}

	processReqLog := ProcessMongoReq{
		Collection: r.col.Name(),
		Method:     "FindOne",
		Query:      filter,
		Document:   nil,
		Options:    nil,
	}

	maskingOption := []logger.MaskingOptionDto{
		{
			MaskingField: "Return.email",
			MaskingType:  logger.Email,
		},
		{
			MaskingField: "Return.first_name",
			MaskingType:  logger.Firstname,
		},
		{
			MaskingField: "Return.last_name",
			MaskingType:  logger.Lastname,
		},
	}

	ctx.Log().Info(logger.NewDBRequest(logger.QUERY, desc), map[string]any{
		"Body": processReqLog,
		"Raw":  processReqLog.RawString(),
	})

	var user UserModel
	err := r.col.FindOne(context.Background(), filter).Decode(&user)
	end := time.Since(start)

	summary := logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "success",
		ResTime:     end.Microseconds(),
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			summary.Code = "404"
			summary.Description = "data_not_found"
		} else {
			summary.Code = "500"
			summary.Description = err.Error()
		}
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.QUERY, err.Error()), map[string]any{
			"Error": err.Error(),
			"Raw":   processReqLog.RawString(),
		})
		return nil, errors.New(summary.Description)
	}

	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.QUERY, desc), map[string]any{
		"Return": user,
	}, maskingOption...)

	uri := "http://localhost:8080"
	user.Href = fmt.Sprintf(uri+"/users/%s", user.ID)

	return &user, nil
}

func (r *userRepository) GetAllUsers(ctx *kp.Context) ([]*UserModel, error) {
	desc := "get all users"
	cmd := "get_all_users"
	node := "mongo"

	start := time.Now()
	filter := map[string]any{
		"deleted_at": primitive.Null{},
	}

	processReqLog := ProcessMongoReq{
		Collection: r.col.Name(),
		Method:     "Find",
		Query:      filter,
		Document:   nil,
		Options:    nil,
	}

	maskingOption := []logger.MaskingOptionDto{
		{
			MaskingField: "Return.*.email",
			MaskingType:  logger.Email,
			IsArray:      true,
		},
		{
			MaskingField: "Return.*.first_name",
			MaskingType:  logger.Firstname,
			IsArray:      true,
		},
		{
			MaskingField: "Return.*.last_name",
			MaskingType:  logger.Lastname,
			IsArray:      true,
		},
	}

	summary := logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "success",
	}

	ctx.Log().Info(logger.NewDBRequest(logger.QUERY, desc), map[string]any{
		"Body": processReqLog,
		"Raw":  processReqLog.RawString(),
	})

	var users []*UserModel
	cursor, err := r.col.Find(context.Background(), filter)
	summary.ResTime = time.Since(start).Microseconds()
	if err != nil {
		summary.Code = "500"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.QUERY, err.Error()), map[string]any{
			"Error": err.Error(),
			"Raw":   processReqLog.RawString(),
		})
		return nil, err
	}
	defer cursor.Close(context.Background())

	uri := "http://localhost:8080"
	for cursor.Next(context.Background()) {
		var user UserModel
		if err := cursor.Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				summary.Code = "404"
				summary.Description = "data_not_found"
			} else {
				summary.Code = "500"
				summary.Description = err.Error()
			}
			ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.QUERY, err.Error()), map[string]any{
				"Error": err.Error(),
				"Raw":   processReqLog.RawString(),
			})
			return nil, err
		}
		user.Href = fmt.Sprintf("%s/users/%s", uri, user.ID)
		users = append(users, &user)
	}

	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.QUERY, desc), map[string]any{
		"Return": users,
	}, maskingOption...)

	return users, nil
}

// func (r *userRepository) UpdateUser(ctx *kp.Context, user *UserModel) error {}

func (r *userRepository) DeleteUser(ctx *kp.Context, id string) error {
	desc := "delete user by id"
	cmd := "delete_user_by_id"
	node := "mongo"

	start := time.Now()

	filter := map[string]interface{}{
		"_id":        id,
		"deleted_at": nil,
	}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"deleted_at": time.Now().UTC(),
		},
	}
	opts := options.Update().SetUpsert(false)
	processReqLog := ProcessMongoReq{
		Collection: r.col.Name(),
		Method:     "UpdateOne",
		Query:      filter,
		Document:   update,
		Options:    opts,
	}

	ctx.Log().Info(logger.NewDBRequest(logger.DELETE, desc), map[string]any{
		"Body": processReqLog,
		"Raw":  processReqLog.RawString(),
	})

	result, err := r.col.UpdateOne(context.Background(), filter, update, opts)
	end := time.Since(start)

	summary := logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "success",
		ResTime:     end.Microseconds(),
	}
	if err != nil {
		summary.Code = "500"
		summary.Description = err.Error()
		ctx.Log().SetSummary(summary).Error(logger.NewDBResponse(logger.DELETE, err.Error()), map[string]any{
			"error": err.Error(),
		})
		return err
	}

	ctx.Log().SetSummary(summary).Info(logger.NewDBResponse(logger.DELETE, desc), map[string]any{
		"Return": result,
	})

	return nil
}
