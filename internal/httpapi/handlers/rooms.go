package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type RoomHandler struct {
	rooms     *repo.RoomRepo
	props     *repo.PropertyRepo
	kinds     *repo.RoomKindRepo
	photos    *repo.PhotoRepo
	photosDir string
}

func NewRoomHandler(r *repo.RoomRepo, p *repo.PropertyRepo, k *repo.RoomKindRepo, ph *repo.PhotoRepo, photosDir string) *RoomHandler {
	return &RoomHandler{rooms: r, props: p, kinds: k, photos: ph, photosDir: photosDir}
}

func (h *RoomHandler) List(c *gin.Context) {
	var propertyID, kindID *int64
	var floor *int
	if v := c.Query("property"); v != "" {
		if p, err := h.props.GetBySlug(c.Request.Context(), v); err == nil {
			propertyID = &p.ID
		}
	}
	if v := c.Query("kind"); v != "" && propertyID != nil {
		if k, err := h.kinds.GetByPropertySlug(c.Request.Context(), c.Query("property"), v); err == nil {
			kindID = &k.ID
		}
	}
	if v := c.Query("floor"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			floor = &n
		}
	}
	rooms, err := h.rooms.List(c.Request.Context(), propertyID, kindID, floor)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	ids := make([]int64, 0, len(rooms))
	for _, r := range rooms {
		ids = append(ids, r.ID)
	}
	pm, _ := h.photos.ListByRoomIDs(c.Request.Context(), ids)
	propsByID, kindsByID := loadPropsAndKinds(c, h.props, h.kinds, rooms)
	out := make([]interface{}, 0, len(rooms))
	for _, r := range rooms {
		out = append(out, toRoomResp(r, pm[r.ID], propsByID[r.PropertyID], kindsByID[r.KindID]))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *RoomHandler) UploadPhotos(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	if _, err := h.rooms.GetByID(c.Request.Context(), id); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_form", "message": err.Error()}})
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		files = form.File["files"]
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "no_files", "message": "no files in form field 'file'"}})
		return
	}

	dir := filepath.Join(h.photosDir, fmt.Sprintf("room_%d", id))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	existing, _ := h.photos.ListByRoom(c.Request.Context(), id)
	pos := len(existing)

	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedExt(ext) {
			continue
		}
		name := randomName() + ext
		dst := filepath.Join(dir, name)
		if err := saveUpload(fh, dst); err != nil {
			middleware.MapDomainError(c, err)
			return
		}
		if _, err := h.photos.Insert(c.Request.Context(), id, name, pos, false); err != nil {
			middleware.MapDomainError(c, err)
			return
		}
		pos++
	}

	all, _ := h.photos.ListByRoom(c.Request.Context(), id)
	out := make([]interface{}, 0, len(all))
	for _, p := range all {
		out = append(out, toPhotoResp(p))
	}
	c.JSON(http.StatusCreated, gin.H{"items": out})
}

func (h *RoomHandler) DeletePhoto(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	photoID, err := strconv.ParseInt(c.Param("photoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	photo, err := h.photos.GetByID(c.Request.Context(), photoID)
	if err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	if photo.RoomID != roomID {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "mismatch", "message": "photo does not belong to room"}})
		return
	}
	_ = os.Remove(filepath.Join(h.photosDir, fmt.Sprintf("room_%d", roomID), photo.Filename))
	if err := h.photos.Delete(c.Request.Context(), photoID); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RoomHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var body struct {
		Label    *string  `json:"label,omitempty"`
		Short    *string  `json:"shortLabel,omitempty"`
		Floor    *int     `json:"floor,omitempty"`
		Side     *string  `json:"side,omitempty"`
		Area     *float64 `json:"area,omitempty"`
		Orient   *string  `json:"orientation,omitempty"`
		IsActive *bool    `json:"isActive,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	var side, orient *string = body.Side, body.Orient
	if err := h.rooms.Update(c.Request.Context(), id, body.Label, body.Short, body.Floor, &side, &orient, body.IsActive, body.Area); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	r, _ := h.rooms.GetByID(c.Request.Context(), id)
	photos, _ := h.photos.ListByRoom(c.Request.Context(), id)
	prop, _ := h.props.GetByID(c.Request.Context(), r.PropertyID)
	kind, _ := h.kinds.GetByID(c.Request.Context(), r.KindID)
	c.JSON(http.StatusOK, toRoomResp(*r, photos, prop, kind))
}

func (h *RoomHandler) ReorderPhotos(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_id", "message": "invalid id"}})
		return
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_json", "message": err.Error()}})
		return
	}
	if err := h.photos.Reorder(c.Request.Context(), id, body.IDs); err != nil {
		middleware.MapDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func allowedExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	}
	return false
}

func randomName() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func saveUpload(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

var _ = domain.Photo{}
