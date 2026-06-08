package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/config"
	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/httpapi/handlers"
	"github.com/ketn1337/reshka-backend/internal/httpapi/middleware"
	"github.com/ketn1337/reshka-backend/internal/repo"
	"github.com/ketn1337/reshka-backend/internal/service"
)

type Deps struct {
	Cfg *config.Config
	// Repos
	Users     *repo.UserRepo
	Props     *repo.PropertyRepo
	Kinds     *repo.RoomKindRepo
	Rooms     *repo.RoomRepo
	Photos    *repo.PhotoRepo
	Guests    *repo.GuestRepo
	Bookings  *repo.BookingRepo
	Rates     *repo.RateRepo
	// Services
	Auth         *service.AuthService
	Booking      *service.BookingService
	BookingStat  *service.BookingStatusService
	Availability *service.AvailabilityService
	// Handlers
	HAuth         *handlers.AuthHandler
	HProps        *handlers.PropertyHandler
	HBookings     *handlers.BookingHandler
	HAdminChess   *handlers.AdminChessboardHandler
	HAvailability *handlers.AvailabilityHandler
	HGuests       *handlers.GuestHandler
	HRates        *handlers.RateHandler
	HRooms        *handlers.RoomHandler
	HUsers        *handlers.UserHandler
}

// NewRouter собирает маршруты.
func NewRouter(d Deps) *gin.Engine {
	if d.Cfg.AppEnv == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Recover())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS(d.Cfg.CORSAllowedOrigins))

	// Photos: раздаём из STATIC_PHOTOS_DIR по /photos/room_<id>/<filename>
	r.GET("/photos/*filepath", func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		clean := filepath.Clean(rel)
		full := filepath.Join(d.Cfg.StaticPhotosDir, clean)
		absBase, _ := filepath.Abs(d.Cfg.StaticPhotosDir)
		absFull, _ := filepath.Abs(full)
		if !strings.HasPrefix(absFull, absBase) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if _, err := os.Stat(absFull); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(absFull)
	})

	// Health
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	api := r.Group("/api")
	{
		// public auth
		api.POST("/auth/login", d.HAuth.Login)
		api.POST("/auth/logout", d.HAuth.Logout)
		api.GET("/auth/me", middleware.RequireAuth(d.Cfg.JWTSecret), d.HAuth.Me)

		// public catalog
		api.GET("/properties", d.HProps.List)
		api.GET("/properties/:slug", d.HProps.Detail)
		api.GET("/properties/:slug/rooms", d.HProps.Rooms)
		api.GET("/rooms/:id", d.HProps.Room)
		api.GET("/availability", d.HAvailability.Search)

		// public booking create
		api.POST("/public/bookings", d.HBookings.Create)
	}

	admin := r.Group("/api/admin", middleware.RequireAuth(d.Cfg.JWTSecret))
	{
		admin.GET("/bookings", d.HBookings.List)
		admin.GET("/bookings/:id", d.HBookings.Detail)
		admin.POST("/bookings", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager, domain.RoleReceptionist), d.HBookings.Create)
		admin.PATCH("/bookings/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager, domain.RoleReceptionist), d.HBookings.Update)
		admin.POST("/bookings/:id/status", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager, domain.RoleReceptionist), d.HBookings.ChangeStatus)
		admin.DELETE("/bookings/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HBookings.Cancel)

		admin.GET("/rooms", d.HRooms.List)
		admin.PATCH("/rooms/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HRooms.Update)
		admin.POST("/rooms/:id/photos", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager, domain.RoleReceptionist), d.HRooms.UploadPhotos)
		admin.DELETE("/rooms/:id/photos/:photoId", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HRooms.DeletePhoto)
		admin.PATCH("/rooms/:id/photos/reorder", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HRooms.ReorderPhotos)

		admin.GET("/chessboard", d.HAdminChess.Get)

		admin.GET("/guests", d.HGuests.Search)
		admin.POST("/guests", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager, domain.RoleReceptionist), d.HGuests.Create)
		admin.PATCH("/guests/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HGuests.Update)

		admin.GET("/rates", d.HRates.List)
		admin.POST("/rates", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HRates.Create)
		admin.PUT("/rates/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HRates.Update)
		admin.DELETE("/rates/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleManager), d.HRates.Delete)

		admin.GET("/users", middleware.RequireRole(domain.RoleAdmin), d.HUsers.List)
		admin.POST("/users", middleware.RequireRole(domain.RoleAdmin), d.HUsers.Create)
		admin.PATCH("/users/:id", middleware.RequireRole(domain.RoleAdmin), d.HUsers.Update)
	}

	return r
}
