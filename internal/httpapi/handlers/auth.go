package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/auth"
	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler { return &AuthHandler{svc: s} }

// Login godoc
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	tok, u, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	middleware.SetAuthCookie(c, tok, 24*3600)
	c.JSON(http.StatusOK, gin.H{
		"token": tok,
		"user": dto.UserResponse{
			ID:       u.ID,
			Email:    u.Email,
			Role:     u.Role,
			FullName: u.FullName,
		},
	})
}

// Logout godoc
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	middleware.ClearAuthCookie(c)
	c.Status(http.StatusNoContent)
}

// Me godoc
// GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	uid, ok := auth.UserIDFromCtx(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	u, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.UserResponse{
		ID:       u.ID,
		Email:    u.Email,
		Role:     u.Role,
		FullName: u.FullName,
	})
}
