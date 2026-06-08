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

type PropertyHandler struct {
	props  *repo.PropertyRepo
	kinds  *repo.RoomKindRepo
	rooms  *repo.RoomRepo
	photos *repo.PhotoRepo
}

func NewPropertyHandler(p *repo.PropertyRepo, k *repo.RoomKindRepo, r *repo.RoomRepo, ph *repo.PhotoRepo) *PropertyHandler {
	return &PropertyHandler{props: p, kinds: k, rooms: r, photos: ph}
}

func (h *PropertyHandler) List(c *gin.Context) {
	props, err := h.props.List(c.Request.Context())
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	out := make([]dto.PropertyResponse, 0, len(props))
	for _, p := range props {
		out = append(out, toPropertyResp(p))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *PropertyHandler) Detail(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.props.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	kinds, err := h.kinds.ListByProperty(c.Request.Context(), p.ID)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	kindOut := make([]dto.RoomKindResponse, 0, len(kinds))
	for _, k := range kinds {
		kindOut = append(kindOut, toKindResp(k))
	}
	c.JSON(http.StatusOK, gin.H{
		"property": toPropertyResp(*p),
		"kinds":    kindOut,
	})
}

func (h *PropertyHandler) Rooms(c *gin.Context) {
	slug := c.Param("slug")
	kindSlug := c.Query("kind")
	p, err := h.props.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	var kindID *int64
	if kindSlug != "" && kindSlug != "any" {
		k, err := h.kinds.GetByPropertySlug(c.Request.Context(), slug, kindSlug)
		if err != nil {
			middleware.MapDomainError(c, err)
			return
		}
		kindID = &k.ID
	}
	rooms, err := h.rooms.List(c.Request.Context(), &p.ID, kindID, nil)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	ids := make([]int64, 0, len(rooms))
	for _, r := range rooms {
		ids = append(ids, r.ID)
	}
	photoMap, _ := h.photos.ListByRoomIDs(c.Request.Context(), ids)
	propsByID, kindsByID := loadPropsAndKinds(c.Request.Context(), h.props, h.kinds, rooms)
	out := make([]dto.RoomResponse, 0, len(rooms))
	for _, r := range rooms {
		out = append(out, toRoomResp(r, photoMap[r.ID], propsByID[r.PropertyID], kindsByID[r.KindID]))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *PropertyHandler) Room(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	r, err := h.rooms.GetByID(c.Request.Context(), id)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	photos, _ := h.photos.ListByRoom(c.Request.Context(), r.ID)
	prop, _ := h.props.GetByID(c.Request.Context(), r.PropertyID)
	kind, _ := h.kinds.GetByID(c.Request.Context(), r.KindID)
	c.JSON(http.StatusOK, toRoomResp(*r, photos, prop, kind))
}

// kinds в ListByProperty отдает []domain.RoomKind
var _ = domain.RoomKind{}
