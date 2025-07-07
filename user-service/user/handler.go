package user

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/sing3demons/go-common-kp/kp/pkg/kp"
	"github.com/sing3demons/go-common-kp/kp/pkg/logger"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) CreateUser(ctx *kp.Context) error {
	var body UserModel

	summary := logger.LogEventTag{
		Node:        "client",
		Command:     "create_user",
		Code:        "200",
		Description: "",
	}

	if err := ctx.Bind(&body); err != nil {
		summary.Code = "400"
		summary.Description = "invalid_request"
		ctx.Log().SetSummary(summary)
		return ctx.JSON(400, map[string]string{
			"error": "invalid_request",
		})
	}

	maskingOption := []logger.MaskingOptionDto{
		{
			MaskingField: "Body.email",
			MaskingType:  logger.Email,
		},
		{
			MaskingField: "Body.first_name",
			MaskingType:  logger.Firstname,
		},
		{
			MaskingField: "Body.last_name",
			MaskingType:  logger.Lastname,
		},
	}

	ctx.Log().SetSummary(summary).Info(logger.NewInbound("create_user", ""), map[string]any{
		"Body":    body,
		"Headers": ctx.Header(),
	}, maskingOption...)

	if err := h.svc.CreateUser(ctx, &body); err != nil {
		return ctx.JSON(500, map[string]string{
			"error": "internal_server",
		})
	}

	ctx.Header().Set("x-rid", ctx.RequestId())
	return ctx.JSON(201, map[string]string{
		"message": "create_success",
		"user_id": body.ID,
	})
}

func (h *Handler) GetUserByID(ctx *kp.Context) error {
	id := ctx.PathParam("id")
	node := "client"
	cmd := "get_user_by_id"
	summary := logger.EventTag(node, cmd, "200", "")
	if id == "" {
		summary.Code = "400"
		summary.Description = "invalid_request"
		ctx.Log().SetSummary(summary).Error(logger.NewInbound(cmd, ""), map[string]any{
			"error": "invalid_user_id",
		})
		ctx.Header().Set("x-rid", ctx.RequestId())
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid_user_id",
		})
	}

	ctx.Log().SetSummary(summary).Info(logger.NewInbound(cmd, ""), map[string]any{
		"Param": map[string]string{
			"key":   "id",
			"value": id,
		},
	})

	user, err := h.svc.GetUserByID(ctx, id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "data_not_found",
		})
	}
	ctx.Header().Set("x-rid", ctx.RequestId())
	return ctx.JSON(http.StatusOK, user)
}

func (h *Handler) GetAllUsers(ctx *kp.Context) error {
	node := "client"
	cmd := "get_all_users"
	summary := logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "",
	}

	ctx.Log().SetSummary(summary).Info(logger.NewInbound(cmd, ""), nil)

	users, err := h.svc.GetAllUsers(ctx)
	if err != nil {
		return ctx.JSON(500, map[string]string{
			"error": "internal_server_error",
		})
	}
	ctx.Header().Set("x-rid", ctx.RequestId())
	return ctx.JSON(http.StatusOK, users)
}

func (h *Handler) DeleteUser(ctx *kp.Context) error {
	id := ctx.PathParam("id")
	node := "client"
	cmd := "delete_user"
	summary := logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "",
	}

	if id == "" {
		summary.Code = "400"
		summary.Description = "invalid_request"
		ctx.Log().SetSummary(summary).Error(logger.NewInbound(cmd, ""), map[string]any{
			"error": "invalid_request",
		})
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid_request",
		})
	}

	ctx.Log().SetSummary(summary).Info(logger.NewInbound(cmd, ""), map[string]any{
		"Param": map[string]string{
			"key":   "id",
			"value": id,
		},
	})

	if err := h.svc.DeleteUser(ctx, id); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "data_not_found",
		})
	}

	ctx.Header().Set("x-rid", ctx.RequestId())
	return ctx.JSON(http.StatusNoContent, map[string]string{
		"message": "delete_success",
	})
}
func (h *Handler) validateUsernameAndEmail(key, value string) error {
	if key == "" || value == "" || (key != "email" && key != "username") {
		return errors.New("invalid_request")
	}
	if key == "email" {
		// validate email format
		emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		matched, err := regexp.MatchString(emailRegex, value)
		if err != nil || !matched {
			return errors.New("invalid_email_format")
		}
	} else if key == "username" {
		// validate username format (e.g., alphanumeric, length)
		usernameRegex := `^[a-zA-Z0-9_]{3,20}$`
		matched, err := regexp.MatchString(usernameRegex, value)
		if err != nil || !matched {
			return errors.New("invalid_username_format")
		}
	}
	return nil
}
func (h *Handler) GetUser(ctx *kp.Context) error {
	key := ctx.PathParam("key")
	value := ctx.PathParam("value")
	node := "client"
	cmd := "get_user"
	summary := logger.LogEventTag{
		Node:        node,
		Command:     cmd,
		Code:        "200",
		Description: "",
	}

	if err := h.validateUsernameAndEmail(key, value); err != nil {
		summary.Code = "400"
		summary.Description = err.Error()

		ctx.Log().SetSummary(summary).Error(logger.NewInbound(cmd, ""), map[string]any{
			"Params": map[string]string{
				"key":   key,
				"value": value,
			},
		})
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid_request",
		})
	}

	ctx.Log().SetSummary(summary).Info(logger.NewInbound(cmd, ""), map[string]any{
		"Params": map[string]string{
			"key":   key,
			"value": value,
		},
	})

	user, err := h.svc.GetUser(ctx, key, value)
	if err != nil {
		if err.Error() == "data_not_found" {
			return ctx.JSON(http.StatusNotFound, map[string]string{
				"error": "data_not_found",
			})
		} else {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
		}
	}

	ctx.Header().Set("x-rid", ctx.RequestId())
	return ctx.JSON(http.StatusOK, user)
}
