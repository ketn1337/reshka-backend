package main

// seed-photos копирует фотографии из reshka-frontend/src/photos в
// ./static/photos/room_<id>/<NN>.jpg и регистрирует их в БД.
//
// Реальная структура исходных фото (на 2026-06):
//   <frontend>/src/photos/<PropertyTitle (roomN)>/01.jpg, 02.jpg, ...
// Например: "Аллея Труда 21 (1)/", "Пионерская 63 (12)/".
//
// Запускается после make seed.

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/joho/godotenv"
	"github.com/ketn1337/reshka-backend/internal/config"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

var folderRe = regexp.MustCompile(`^(?P<title>.+?)\s*\((?P<n>\d+)\)\s*$`)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.StaticPhotosDir == "" {
		log.Fatal("STATIC_PHOTOS_DIR is empty")
	}
	srcRoot := os.Getenv("FRONTEND_PHOTOS_DIR")
	if srcRoot == "" {
		srcRoot = "../reshka-frontend/src/photos"
	}

	ctx := context.Background()
	db, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	propsR := repo.NewPropertyRepo(db)
	roomsR := repo.NewRoomRepo(db)
	photosR := repo.NewPhotoRepo(db)

	props, err := propsR.List(ctx)
	if err != nil {
		log.Fatalf("props: %v", err)
	}
	if len(props) == 0 {
		log.Fatal("no properties in db, run `make seed` first")
	}

	// Карта: property.slug → [roomID for shortLabel]
	propRooms := make(map[string]map[string]int64, len(props))
	for _, p := range props {
		rooms, err := roomsR.List(ctx, &p.ID, nil, nil)
		if err != nil {
			log.Printf("rooms for %s: %v", p.Slug, err)
			continue
		}
		m := make(map[string]int64, len(rooms))
		for _, r := range rooms {
			m[r.ShortLabel] = r.ID
		}
		propRooms[p.Slug] = m
	}

	// Словарь известных title → slug
	titleToSlug := map[string]string{
		"аллея труда 21": "alley",
		"пионерская 63":  "pioneer",
	}

	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		log.Fatalf("read src: %v", err)
	}

	total := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := folderRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(m[1]))
		shortLabel := m[2]
		slug, ok := titleToSlug[title]
		if !ok {
			log.Printf("unknown property title: %q (folder %s)", title, e.Name())
			continue
		}
		roomMap, ok := propRooms[slug]
		if !ok {
			continue
		}
		roomID, ok := roomMap[shortLabel]
		if !ok {
			log.Printf("no room shortLabel=%s in %s, skipping folder %s", shortLabel, slug, e.Name())
			continue
		}

		files, err := os.ReadDir(filepath.Join(srcRoot, e.Name()))
		if err != nil {
			continue
		}
		pos := 0
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
				continue
			}
			src := filepath.Join(srcRoot, e.Name(), name)
			dst := filepath.Join(cfg.StaticPhotosDir, fmt.Sprintf("room_%d", roomID), name)
			if err := copyFile(src, dst); err != nil {
				log.Printf("copy %s: %v", src, err)
				continue
			}
			if _, err := photosR.Insert(ctx, roomID, name, pos, pos == 0); err != nil {
				log.Printf("db insert room=%d file=%s: %v", roomID, name, err)
			}
			pos++
			total++
		}
	}
	fmt.Printf("seed-photos: copied %d files\n", total)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

var _ = strconv.Itoa
