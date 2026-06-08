package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	zl "github.com/rs/zerolog/log"

	"github.com/ketn1337/reshka-backend/internal/config"
	"github.com/ketn1337/reshka-backend/internal/db"
	"github.com/ketn1337/reshka-backend/internal/httpapi"
	"github.com/ketn1337/reshka-backend/internal/httpapi/handlers"
	"github.com/ketn1337/reshka-backend/internal/repo"
	"github.com/ketn1337/reshka-backend/internal/service"
	"github.com/ketn1337/reshka-backend/migrations"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	zl.Logger = zl.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load .env если есть (best-effort, в проде просто ничего не делаем).
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		zl.Fatal().Err(err).Msg("config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		zl.Fatal().Err(err).Msg("db open")
	}
	defer database.Close()

	if cfg.AppEnv != "prod" {
		if err := db.RunMigrations(database, migrations.FS, "."); err != nil {
			zl.Fatal().Err(err).Msg("migrations")
		}
		zl.Info().Msg("migrations applied")
	}

	// Репозитории
	usersR := repo.NewUserRepo(database)
	propsR := repo.NewPropertyRepo(database)
	kindsR := repo.NewRoomKindRepo(database)
	roomsR := repo.NewRoomRepo(database)
	photosR := repo.NewPhotoRepo(database)
	guestsR := repo.NewGuestRepo(database)
	bookingsR := repo.NewBookingRepo(database)
	ratesR := repo.NewRateRepo(database)

	// Сервисы
	authSvc := service.NewAuthService(usersR, cfg.JWTSecret, time.Duration(cfg.JWTTTLHours)*time.Hour)
	bookSvc := service.NewBookingService(bookingsR, guestsR, roomsR, ratesR, kindsR)
	bookStatSvc := service.NewBookingStatusService(bookingsR)
	availSvc := service.NewAvailabilityService(roomsR, bookingsR, kindsR, propsR, guestsR)

	// Handlers
	hAuth := handlers.NewAuthHandler(authSvc)
	hProps := handlers.NewPropertyHandler(propsR, kindsR, roomsR, photosR)
	hBookings := handlers.NewBookingHandler(bookSvc, bookStatSvc, bookingsR, roomsR, kindsR, propsR, guestsR)
	hChess := handlers.NewAdminChessboardHandler(propsR, kindsR, availSvc)
	hAvail := handlers.NewAvailabilityHandler(propsR, kindsR, availSvc)
	hGuests := handlers.NewGuestHandler(guestsR)
	hRates := handlers.NewRateHandler(ratesR, kindsR)
	hRooms := handlers.NewRoomHandler(roomsR, propsR, kindsR, photosR, cfg.StaticPhotosDir)
	hUsers := handlers.NewUserHandler(usersR)

	router := httpapi.NewRouter(httpapi.Deps{
		Cfg: cfg,
		Users: usersR, Props: propsR, Kinds: kindsR, Rooms: roomsR,
		Photos: photosR, Guests: guestsR, Bookings: bookingsR, Rates: ratesR,
		Auth: authSvc, Booking: bookSvc, BookingStat: bookStatSvc, Availability: availSvc,
		HAuth: hAuth, HProps: hProps, HBookings: hBookings, HAdminChess: hChess,
		HAvailability: hAvail, HGuests: hGuests, HRates: hRates, HRooms: hRooms, HUsers: hUsers,
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.AppPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		zl.Info().Msg("shutting down")
		shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shCancel()
		_ = srv.Shutdown(shCtx)
	}()

	zl.Info().Str("addr", srv.Addr).Msg("listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zl.Fatal().Err(err).Msg("listen")
	}
	_ = zl.Logger
}
