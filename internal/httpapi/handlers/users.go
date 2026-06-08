package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/auth"
	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type UserHandler struct {
	users *repo.UserRepo
}

func NewUserHandler(u *repo.UserRepo) *UserHandler { return &UserHandler{users: u} }

func (h *UserHandler) List(c *gin.Context) {
	us, err := h.users.List(c.Request.Context())
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	out := make([]dto.UserResponse, 0, len(us))
	for _, u := range us {
		out = append(out, dto.UserResponse{ID: u.ID, Email: u.Email, Role: u.Role, FullName: u.FullName})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "weak_password", "message": err.Error()}})
		return
	}
	id, err := h.users.Create(c.Request.Context(), req.Email, hash, req.Role, req.FullName)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	u, _ := h.users.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusCreated, dto.UserResponse{ID: u.ID, Email: u.Email, Role: u.Role, FullName: u.FullName})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	if err := h.users.Update(c.Request.Context(), id, req.Role, req.FullName, req.IsActive); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	if req.Password != nil {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "weak_password", "message": err.Error()}})
			return
		}
		// нет метода UpdatePassword — простой подход: используем Create как upsert с заменой хэша.
		// Для MVP сойдёт: запрашиваем email, обновляем через прямой SQL в repo.
		// Чтобы не плодить ad-hoc, прокинем через SQL-метод:
		if err := h.users.UpdatePassword(c.Request.Context(), id, hash); err != nil {
			middleware.MapDomainError(c, err)
			return
		}
	}
	u, err := h.users.GetByID(c.Request.Context(), id)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.UserResponse{ID: u.ID, Email: u.Email, Role: u.Role, FullName: u.FullName})
}
