package main

// seed идемпотентно наполняет БД базовыми данными:
//   - 2 property (alley, pioneer)
//   - 2 kind для каждого (standard, comfort) с базовой ставкой
//   - 13 комнат для alley + 14 для pioneer
//   - админ-пользователь из env
//   - дефолтный период тарифов на год вперёд
//
// Идемпотентность обеспечивают ON CONFLICT в repo.

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/joho/godotenv"
	"github.com/ketn1337/reshka-backend/internal/auth"
	"github.com/ketn1337/reshka-backend/internal/config"
	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type propertyDef struct {
	Slug, Title, ShortTitle, Address, Accent string
	Kinds                                    []kindDef
}

type kindDef struct {
	Slug, Title, Beds string
	BaseRate          float64
	Capacity          int
	Area              float64
	Rooms             []roomDef
}

type roomDef struct {
	Label, Short string
	Floor        int
	Side         string
	Orientation  string
}

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	db, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	usersR := repo.NewUserRepo(db)
	propsR := repo.NewPropertyRepo(db)
	kindsR := repo.NewRoomKindRepo(db)
	roomsR := repo.NewRoomRepo(db)
	ratesR := repo.NewRateRepo(db)

	props := buildProperties()

	for _, p := range props {
		pid, err := propsR.Upsert(ctx, domain.Property{
			Slug:       p.Slug,
			Title:      p.Title,
			ShortTitle: p.ShortTitle,
			Address:    p.Address,
			Accent:     strPtr(p.Accent),
		})
		if err != nil {
			log.Fatalf("upsert property %s: %v", p.Slug, err)
		}
		fmt.Printf("property %s -> id=%d\n", p.Slug, pid)

		for _, k := range p.Kinds {
			kid, err := kindsR.Upsert(ctx, domain.RoomKind{
				PropertyID: pid,
				Slug:       k.Slug,
				Title:      k.Title,
				BaseRate:   k.BaseRate,
				Capacity:   k.Capacity,
				Area:       k.Area,
				Beds:       k.Beds,
			})
			if err != nil {
				log.Fatalf("upsert kind %s/%s: %v", p.Slug, k.Slug, err)
			}

			for _, r := range k.Rooms {
				orient := r.Orientation
				side := r.Side
				_, err := roomsR.Upsert(ctx, domain.Room{
					PropertyID:  pid,
					KindID:      kid,
					Label:       r.Label,
					ShortLabel:  r.Short,
					Floor:       r.Floor,
					Side:        strPtrIfNotEmpty(side),
					Orientation: strPtrIfNotEmpty(orient),
					IsActive:    true,
				})
				if err != nil {
					log.Fatalf("upsert room %s/%s: %v", p.Slug, r.Short, err)
				}
			}
			fmt.Printf("  kind %s/%s -> id=%d (%d rooms)\n", p.Slug, k.Slug, kid, len(k.Rooms))

			// дефолтный тариф — год вперёд
			now := time.Now().UTC().Truncate(24 * time.Hour)
			from := now
			to := now.AddDate(1, 0, 0)
			_, err = ratesR.Create(ctx, domain.Rate{
				KindID:      kid,
				DateFrom:    from,
				DateTo:      to,
				WeekdayRate: k.BaseRate,
				WeekendRate: k.BaseRate * 1.2,
			})
			if err != nil && !strings.Contains(err.Error(), "conflict") && !strings.Contains(err.Error(), "duplicate") {
				log.Printf("warn rate %s/%s: %v", p.Slug, k.Slug, err)
			}
		}
	}

	// админ
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		hash, err := auth.HashPassword(cfg.AdminPassword)
		if err != nil {
			log.Fatalf("hash: %v", err)
		}
		uid, err := usersR.Create(ctx, cfg.AdminEmail, hash, domain.RoleAdmin, cfg.AdminFullName)
		if err != nil {
			log.Fatalf("admin create: %v", err)
		}
		fmt.Printf("admin id=%d email=%s\n", uid, cfg.AdminEmail)
	}

	fmt.Println("seed: ok")
}

func buildProperties() []propertyDef {
	return []propertyDef{
		{
			Slug: "alley", Title: "Аллея Труда 21", ShortTitle: "Аллея Труда",
			Address: "ул. Аллея Труда, 21, Комсомольск-на-Амуре", Accent: "#f6c90e",
			Kinds: []kindDef{
				{
					Slug: "standard", Title: "Стандарт", Beds: "1 двуспальная",
					BaseRate: 2200, Capacity: 2, Area: 14,
					Rooms: roomsForFloor("Аллея Труда", "standard", 13),
				},
				{
					Slug: "comfort", Title: "Комфорт", Beds: "1 двуспальная + диван",
					BaseRate: 2900, Capacity: 3, Area: 20,
					Rooms: []roomDef{}, // на Аллее только standard
				},
			},
		},
		{
			Slug: "pioneer", Title: "Пионерская 63", ShortTitle: "Пионерская",
			Address: "ул. Пионерская, 63, Комсомольск-на-Амуре", Accent: "#f6c90e",
			Kinds: []kindDef{
				{
					Slug: "standard", Title: "Стандарт", Beds: "1 двуспальная",
					BaseRate: 2200, Capacity: 2, Area: 14,
					Rooms: roomsForFloor("Пионерская", "standard", 14),
				},
				{
					Slug: "comfort", Title: "Комфорт", Beds: "1 двуспальная + диван",
					BaseRate: 2900, Capacity: 3, Area: 20,
					Rooms: []roomDef{},
				},
			},
		},
	}
}

func roomsForFloor(_ string, _ string, total int) []roomDef {
	out := make([]roomDef, 0, total)
	for i := 1; i <= total; i++ {
		floor := (i-1)/7 + 1
		side := "A"
		if i%2 == 0 {
			side = "Б"
		}
		out = append(out, roomDef{
			Label:      fmt.Sprintf("Номер %d", i),
			Short:      fmt.Sprintf("%d", i),
			Floor:      floor,
			Side:       side,
			Orientation: orientFor(i),
		})
	}
	return out
}

func orientFor(i int) string {
	switch {
	case i%4 == 0:
		return domain.OrientationStreet
	case i%4 == 1:
		return domain.OrientationInner
	case i%4 == 2:
		return domain.OrientationCourtyard
	default:
		return domain.OrientationInner
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ = os.Getenv
