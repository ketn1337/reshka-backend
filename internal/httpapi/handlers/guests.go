package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type GuestHandler struct {
	guests *repo.GuestRepo
}

func NewGuestHandler(g *repo.GuestRepo) *GuestHandler { return &GuestHandler{guests: g} }

func (h *GuestHandler) Search(c *gin.Context) {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	gs, err := h.guests.Search(c.Request.Context(), q, limit)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	out := make([]dto.GuestResponse, 0, len(gs))
	for _, g := range gs {
		out = append(out, toGuestResp(g))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *GuestHandler) Create(c *gin.Context) {
	var req dto.GuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	id, err := h.guests.Create(c.Request.Context(), guestFromReq(req))
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	g, _ := h.guests.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusCreated, toGuestResp(*g))
}

func (h *GuestHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var req dto.GuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	if err := h.guests.Update(c.Request.Context(), id, guestFromReq(req)); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	g, _ := h.guests.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, toGuestResp(*g))
}

func guestFromReq(r dto.GuestRequest) domain.Guest {
	return domain.Guest{
		FullName:  r.FullName,
		Phone:     r.Phone,
		Email:     r.Email,
		DocType:   r.DocType,
		DocNumber: r.DocNumber,
		Notes:     r.Notes,
	}
}
