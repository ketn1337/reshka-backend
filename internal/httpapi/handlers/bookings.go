package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/auth"
	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
	"github.com/ketn1337/reshka-backend/internal/service"
)

type BookingHandler struct {
	svc       *service.BookingService
	statusSvc *service.BookingStatusService
	bookings  *repo.BookingRepo
	rooms     *repo.RoomRepo
	roomKinds *repo.RoomKindRepo
	props     *repo.PropertyRepo
	guests    *repo.GuestRepo
}

func NewBookingHandler(
	s *service.BookingService, ss *service.BookingStatusService,
	b *repo.BookingRepo, r *repo.RoomRepo, rk *repo.RoomKindRepo,
	p *repo.PropertyRepo, g *repo.GuestRepo,
) *BookingHandler {
	return &BookingHandler{
		svc: s, statusSvc: ss, bookings: b, rooms: r, roomKinds: rk, props: p, guests: g,
	}
}

func (h *BookingHandler) List(c *gin.Context) {
	q := c.Request.URL.Query()
	from := parseDateOpt(q.Get("from"))
	to := parseDateOpt(q.Get("to"))
	status := strPtr(q.Get("status"))
	search := strPtr(q.Get("q"))

	var propertyID, kindID *int64
	if v := q.Get("property"); v != "" {
		if p, err := h.props.GetBySlug(c.Request.Context(), v); err == nil {
			propertyID = &p.ID
		}
	}
	if v := q.Get("kind"); v != "" && propertyID != nil {
		if k, err := h.roomKinds.GetByPropertySlug(c.Request.Context(), q.Get("property"), v); err == nil {
			kindID = &k.ID
		}
	}

	bs, err := h.bookings.List(c.Request.Context(), from, to, propertyID, kindID, status, search)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	out := make([]dto.BookingResponse, 0, len(bs))
	for _, b := range bs {
		out = append(out, h.toResp(c, b))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *BookingHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	b, err := h.bookings.GetByID(c.Request.Context(), id)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toResp(c, *b))
}

func (h *BookingHandler) Create(c *gin.Context) {
	var req dto.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	ci, err := time.Parse("2006-01-02", req.CheckIn)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "checkIn"}})
		return
	}
	co, err := time.Parse("2006-01-02", req.CheckOut)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "checkOut"}})
		return
	}
	checkInTime := normalizeHM(req.CheckInTime, "14:00:00")
	checkOutTime := normalizeHM(req.CheckOutTime, "12:00:00")
	uid, _ := auth.UserIDFromCtx(c)
	var gi *service.GuestInput
	if req.Guest != nil {
		gi = &service.GuestInput{
			FullName:  req.Guest.FullName,
			Phone:     req.Guest.Phone,
			Email:     req.Guest.Email,
			DocType:   req.Guest.DocType,
			DocNumber: req.Guest.DocNumber,
			Notes:     req.Guest.Notes,
		}
	}
	b, err := h.svc.Create(c.Request.Context(), service.CreateBookingInput{
		RoomID:       req.RoomID,
		CheckIn:      ci,
		CheckOut:     co,
		CheckInTime:  checkInTime,
		CheckOutTime: checkOutTime,
		Adults:       req.Adults,
		Source:       req.Source,
		GuestID:      req.GuestID,
		Guest:        gi,
		Total:        req.Total,
		Prepay:       req.Prepay,
		Notes:        req.Notes,
		CreatedBy:    uid,
	})
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toResp(c, *b))
}

func (h *BookingHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var req dto.UpdateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	var ci, co *time.Time
	if req.CheckIn != nil {
		t, err := time.Parse("2006-01-02", *req.CheckIn)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "checkIn"}})
			return
		}
		ci = &t
	}
	if req.CheckOut != nil {
		t, err := time.Parse("2006-01-02", *req.CheckOut)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_date", "message": "checkOut"}})
			return
		}
		co = &t
	}
	if err := h.bookings.UpdateFields(c.Request.Context(), id, ci, co,
		normalizeHMOpt(req.CheckInTime), normalizeHMOpt(req.CheckOutTime),
		req.Adults, req.GuestID, req.Total, req.Prepayment, req.Notes); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	b, err := h.bookings.GetByID(c.Request.Context(), id)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toResp(c, *b))
}

func (h *BookingHandler) ChangeStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var req dto.ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	uid, _ := auth.UserIDFromCtx(c)
	if err := h.statusSvc.Change(c.Request.Context(), id, req.To, uid, req.Reason); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	b, _ := h.bookings.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, h.toResp(c, *b))
}

func (h *BookingHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	uid, _ := auth.UserIDFromCtx(c)
	reason := "Отменено администратором"
	if err := h.statusSvc.Change(c.Request.Context(), id, domain.BookingStatusCancelled, uid, &reason); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	b, _ := h.bookings.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, h.toResp(c, *b))
}

func (h *BookingHandler) toResp(c *gin.Context, b domain.Booking) dto.BookingResponse {
	room, _ := h.rooms.GetByID(c.Request.Context(), b.RoomID)
	var prop *domain.Property
	if room != nil {
		prop, _ = h.props.GetByID(c.Request.Context(), room.PropertyID)
	}
	var guest *domain.Guest
	if b.GuestID != nil {
		guest, _ = h.guests.GetByID(c.Request.Context(), *b.GuestID)
	}
	hist, _ := h.bookings.History(c.Request.Context(), b.ID)
	return toBookingResp(b, room, prop, guest, hist)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseDateOpt(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// normalizeHMOpt: nil → nil; "" → nil; "HH:MM" → "HH:MM:SS"; иначе as-is.
func normalizeHMOpt(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	v := *s
	if len(v) == 5 {
		v = v + ":00"
	}
	return &v
}
