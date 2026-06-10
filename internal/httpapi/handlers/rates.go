package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type RateHandler struct {
	rates *repo.RateRepo
	kinds *repo.RoomKindRepo
}

func NewRateHandler(r *repo.RateRepo, k *repo.RoomKindRepo) *RateHandler {
	return &RateHandler{rates: r, kinds: k}
}

func (h *RateHandler) List(c *gin.Context) {
	kindIDStr := c.Query("kindId")
	propertySlug := c.Query("property")
	kindSlug := c.Query("kind")

	var kindID int64
	if kindIDStr != "" {
		id, err := strconv.ParseInt(kindIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "kindId"}})
			return
		}
		kindID = id
	} else if propertySlug != "" && kindSlug != "" {
		k, err := h.kinds.GetByPropertySlug(c.Request.Context(), propertySlug, kindSlug)
		if err != nil {
			middleware.MapDomainError(c, err)
			return
		}
		kindID = k.ID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_request", "message": "kindId or property+kind required"}})
		return
	}

	rs, err := h.rates.ListByKind(c.Request.Context(), kindID)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	out := make([]dto.RateResponse, 0, len(rs))
	for _, x := range rs {
		out = append(out, toRateResp(x))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *RateHandler) Create(c *gin.Context) {
	var req dto.RateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	df, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "dateFrom"}})
		return
	}
	dt, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "dateTo"}})
		return
	}
	if dt.Before(df) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_range", "message": "dateTo before dateFrom"}})
		return
	}
	x := domain.Rate{
		KindID:      req.KindID,
		DateFrom:    df,
		DateTo:      dt,
		WeekdayRate: req.WeekdayRate,
		WeekendRate: req.WeekendRate,
	}
	id, err := h.rates.Create(c.Request.Context(), x)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	saved, _ := h.rates.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusCreated, toRateResp(*saved))
}

func (h *RateHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var req dto.RateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	df, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "dateFrom"}})
		return
	}
	dt, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "dateTo"}})
		return
	}
	x := domain.Rate{
		ID:          id,
		KindID:      req.KindID,
		DateFrom:    df,
		DateTo:      dt,
		WeekdayRate: req.WeekdayRate,
		WeekendRate: req.WeekendRate,
	}
	if err := h.rates.Update(c.Request.Context(), x); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	saved, _ := h.rates.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, toRateResp(*saved))
}

func (h *RateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	if err := h.rates.Delete(c.Request.Context(), id); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
