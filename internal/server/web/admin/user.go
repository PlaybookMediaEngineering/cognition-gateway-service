package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/user"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
)

type UserManager interface {
	GetUsers(tags, keyIds, userIds []string, offset int, limit int) ([]*user.User, error)
	CreateUser(u *user.User) (*user.User, error)
	UpdateUser(id string, uu *user.UpdateUser) (*user.User, error)
	UpdateUserViaTagsAndUserId(tags []string, uid string, uu *user.UpdateUser) (*user.User, error)
}

func getGetUsersHandler(m UserManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_get_users_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_get_users_handler.latency", dur, nil, 1)
		}()

		path := "/api/users"

		tags := c.QueryArray("tags")
		keyIds := c.QueryArray("keyIds")
		userIds := c.QueryArray("userIds")

		offset := 0
		offsetStr, ok := c.GetQuery("offset")

		if ok {
			parsed, err := strconv.Atoi(offsetStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/bad-filters",
					Title:    "bad offset query param",
					Status:   http.StatusBadRequest,
					Detail:   "offset query param cannot be converted to integer",
					Instance: path,
				})
				return
			}

			offset = parsed
		}

		limit := 0
		limitStr, ok := c.GetQuery("limit")
		if ok {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/bad-filters",
					Title:    "bad limit query param",
					Status:   http.StatusBadRequest,
					Detail:   "limit query param cannot be converted to integer",
					Instance: path,
				})
				return
			}

			limit = parsed
		}

		if len(tags) == 0 && len(userIds) == 0 && len(keyIds) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-filteres",
				Title:    "filters are not found",
				Status:   http.StatusBadRequest,
				Detail:   "filters are missing from the request url. it is required for retrieving users.",
				Instance: path,
			})
			return
		}

		keys, err := m.GetUsers(tags, keyIds, userIds, offset, limit)
		if err != nil {
			telemetry.Incr("bricksllm.admin.get_get_users_handler.get_users_err", nil, 1)

			logError(log, "error when getting api keys by tag", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/getting-keys",
				Title:    "getting keys errored out",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_get_users_handler.success", nil, 1)
		c.JSON(http.StatusOK, keys)
	}
}

func getCreateUserHandler(m UserManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_create_user_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_create_user_handler.latency", dur, nil, 1)
		}()

		path := "/api/users"
		if c == nil || c.Request == nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/empty-context",
				Title:    "context is empty error",
				Status:   http.StatusInternalServerError,
				Detail:   "gin context is empty",
				Instance: path,
			})
			return
		}

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading user creation request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		u := &user.User{}
		err = json.Unmarshal(data, u)
		if err != nil {
			logError(log, "error when unmarshalling user creation request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		created, err := m.CreateUser(u)
		if err != nil {
			errType := "internal"

			defer func() {
				telemetry.Incr("bricksllm.admin.get_create_user_handler.create_key_error", []string{
					"error_type:" + errType,
				}, 1)
			}()

			if _, ok := err.(validationError); ok {
				errType = "validation"

				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "create user validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			logError(log, "error when creating user", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/user-manager",
				Title:    "user creation error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_create_user_handler.success", nil, 1)

		c.JSON(http.StatusOK, created)
	}
}

func getUpdateUserHandler(m UserManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_update_user_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_update_user_handler.latency", dur, nil, 1)
		}()

		path := "/api/users/:id"
		if c == nil || c.Request == nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/empty-context",
				Title:    "context is empty error",
				Status:   http.StatusInternalServerError,
				Detail:   "gin context is empty",
				Instance: path,
			})
			return
		}

		uid := c.Param("id")
		if len(uid) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-user-id",
				Title:    "missing user id",
				Status:   http.StatusBadRequest,
				Detail:   "user id is empty",
				Instance: path,
			})
			return
		}

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading update user request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		uu := &user.UpdateUser{}
		err = json.Unmarshal(data, uu)
		if err != nil {
			logError(log, "error when unmarshalling update user request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		resk, err := m.UpdateUser(uid, uu)
		if err != nil {
			errType := "internal"

			defer func() {
				telemetry.Incr("bricksllm.admin.get_update_user_handler.create_key_error", []string{
					"error_type:" + errType,
				}, 1)
			}()

			if _, ok := err.(validationError); ok {
				errType = "validation"

				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "update user validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			logError(log, "error when updating user", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/user-manager",
				Title:    "update user error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_update_user_handler.success", nil, 1)

		c.JSON(http.StatusOK, resk)
	}
}

func getUpdateUserViaTagsAndUserIdHandler(m UserManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_update_user_via_tags_and_user_id_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_update_user_via_tags_and_user_id_handler.latency", dur, nil, 1)
		}()

		path := "/api/users"
		if c == nil || c.Request == nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/empty-context",
				Title:    "context is empty error",
				Status:   http.StatusInternalServerError,
				Detail:   "gin context is empty",
				Instance: path,
			})
			return
		}

		uid := c.Query("userId")
		if len(uid) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-user-id",
				Title:    "missing user id",
				Status:   http.StatusBadRequest,
				Detail:   "query param tags is empty",
				Instance: path,
			})
			return
		}

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading update user via tags and user id request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		uu := &user.UpdateUser{}
		err = json.Unmarshal(data, uu)
		if err != nil {
			logError(log, "error when unmarshalling update user via tags and user id body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		resk, err := m.UpdateUserViaTagsAndUserId(c.QueryArray("tags"), uid, uu)
		if err != nil {
			errType := "internal"

			defer func() {
				telemetry.Incr("bricksllm.admin.get_update_user_via_tags_and_user_id_handler.create_key_error", []string{
					"error_type:" + errType,
				}, 1)
			}()

			if _, ok := err.(validationError); ok {
				errType = "validation"

				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "update user validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			logError(log, "error when updating user", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/user-manager",
				Title:    "update user error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_update_user_via_tags_and_user_id_handler.success", nil, 1)

		c.JSON(http.StatusOK, resk)
	}
}
