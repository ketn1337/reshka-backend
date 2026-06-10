package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
	"github.com/ketn1337/reshka-backend/internal/service"
)

type AvailabilityHandler struct {
	props *repo.PropertyRepo
	kinds *repo.RoomKindRepo
	svc   *service.AvailabilityService
}

func NewAvailabilityHandler(p *repo.PropertyRepo, k *repo.RoomKindRepo, s *service.AvailabilityService) *AvailabilityHandler {
	return &AvailabilityHandler{props: p, kinds: k, svc: s}
}

// GET /api/availability?property=<slug>&checkIn=YYYY-MM-DD&checkOut=YYYY-MM-DD&kind=<slug>
func (h *AvailabilityHandler) Search(c *gin.Context) {
	propSlug := c.Query("property")
	checkIn := c.Query("checkIn")
	checkOut := c.Query("checkOut")
	kindSlug := c.Query("kind")

	if propSlug == "" || checkIn == "" || checkOut == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_request", "message": "property, checkIn, checkOut required"}})
		return
	}
	ci, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "checkIn"}})
		return
	}
	co, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "checkOut"}})
		return
	}
	if !co.After(ci) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_range", "message": "checkOut must be after checkIn"}})
		return
	}

	p, err := h.props.GetBySlug(c.Request.Context(), propSlug)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	var kindID *int64
	if kindSlug != "" && kindSlug != "any" {
		k, err := h.kinds.GetByPropertySlug(c.Request.Context(), propSlug, kindSlug)
		if err != nil {
			middleware.MapDomainError(c, err)
			return
		}
		kindID = &k.ID
	}
	// при kindID==nil — все kinds
	var pid int64 = p.ID
	var rows []service.AvailabilityRoom
	if kindID == nil {
		rows, err = h.svc.SearchAll(c.Request.Context(), pid, ci, co)
	} else {
		rows, err = h.svc.Search(c.Request.Context(), pid, *kindID, ci, co)
	}
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func ptrOrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// Chessboard godoc
// GET /api/admin/chessboard?from=YYYY-MM-DD&days=14
// Показывает все номера обоих объектов; property/kind фильтры убраны (decision пользователя).
type AdminChessboardHandler struct {
	props *repo.PropertyRepo
	kinds *repo.RoomKindRepo
	svc   *service.AvailabilityService
}

func NewAdminChessboardHandler(p *repo.PropertyRepo, k *repo.RoomKindRepo, s *service.AvailabilityService) *AdminChessboardHandler {
	return &AdminChessboardHandler{props: p, kinds: k, svc: s}
}

func (h *AdminChessboardHandler) Get(c *gin.Context) {
	from := c.Query("from")
	daysStr := c.DefaultQuery("days", "14")

	if from == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_request", "message": "from required"}})
		return
	}
	fromT, err := time.Parse("2006-01-02", from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "from"}})
		return
	}
	days, _ := strconv.Atoi(daysStr)
	res, err := h.svc.Chessboard(c.Request.Context(), fromT, days)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	// Конвертим в DTO-ответ (rooms с propertyTitle/kindTitle)
	roomsOut := make([]dto.RoomResponse, 0, len(res.Rooms))
	propsByID, kindsByID := loadPropsAndKinds(c.Request.Context(), h.props, h.kinds, res.Rooms)
	for _, r := range res.Rooms {
		roomsOut = append(roomsOut, toRoomResp(r, nil, propsByID[r.PropertyID], kindsByID[r.KindID]))
	}
	barsOut := make([]dto.ChessboardBar, 0, len(res.Bookings))
	for _, b := range res.Bookings {
		barsOut = append(barsOut, dto.ChessboardBar{
			BookingID:   b.BookingID,
			RoomID:      b.RoomID,
			Code:        b.Code,
			StartISO:    b.StartISO,
			EndISO:      b.EndISO,
			Nights:      b.Nights,
			Status:      b.Status,
			GuestName:   b.GuestName,
			Adults:      b.Adults,
			Source:      b.Source,
			TotalAmount: b.TotalAmount,
		})
	}
	c.JSON(http.StatusOK, dto.ChessboardResult{
		Rooms:    roomsOut,
		Days:     res.Days,
		Bookings: barsOut,
	})
}
