// import-bnovo — разовый импорт броней из Bnovo PMS в нашу БД.
//
// Подкоманды:
//   --mode=discover   дампит уникальные пары (bnovo_room_id, room_type_name) из всех
//                     броней в окне date_from..date_to, чтобы заполнить bnovo-rooms.json
//   --mode=wipe       удаляет все брони + гостей (с бэкапом EXCLUDE)
//   --mode=import     тянет брони из Bnovo и создаёт локальные
//   --mode=all        wipe + import одной программой
//
// Флаги:
//   --config=PATH     путь к bnovo-rooms.json (обяз.)
//   --dry-run         только печать, БД не трогаем
//   --limit=N         обработать не более N броней (0 = все); для отладки
//
// Идемпотентность: import пропускает бронь, если (bnovo_id, room_id) уже есть.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"

	"github.com/ketn1337/reshka-backend/internal/bnovo"
	"github.com/ketn1337/reshka-backend/internal/config"
	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/repo"
	"github.com/ketn1337/reshka-backend/internal/service"
)

// =========================
// Конфиг
// =========================

type roomMapping struct {
	BnovoRoomID   int64  `json:"bnovoRoomId"`
	BnovoRoomName string `json:"bnovoRoomName"`
	OurRoomID     int64  `json:"ourRoomId"`
}

type importConfig struct {
	BnovoAccountID int64         `json:"bnovoAccountId"`
	BnovoAPIKey    string        `json:"bnovoApiKey"`
	DateFrom       string        `json:"dateFrom"` // "YYYY-MM-DD"
	DateTo         string        `json:"dateTo"`
	// PropertyPrefixes — белый список префиксов адресов, которые нас интересуют.
	// Если пусто — импортируем ВСЕ комнаты из Bnovo (опасно, если в аккаунте
	// несколько объектов, как в нашем случае: Дмитровское, Смольная, Пионерская…).
	// Рекомендуется заполнить: ["Пионерская 63", "Аллея Труда 21"].
	PropertyPrefixes []string      `json:"propertyPrefixes"`
	Rooms            []roomMapping `json:"rooms"`
}

func loadConfig(path string) (*importConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c importConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if c.BnovoAccountID == 0 || c.BnovoAPIKey == "" {
		return nil, errors.New("config: bnovoAccountId и bnovoApiKey обязательны")
	}
	if c.DateFrom == "" || c.DateTo == "" {
		return nil, errors.New("config: dateFrom и dateTo обязательны")
	}
	return &c, nil
}

// loadConfigWipeOnly — облегчённая загрузка для mode=wipe, когда Bnovo-креды
// не нужны. DateFrom/DateTo тоже не проверяем.
func loadConfigWipeOnly(path string) (*importConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c importConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &c, nil
}

// =========================
// Утилиты
// =========================

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func nextDay(t time.Time) time.Time {
	return t.AddDate(0, 0, 1)
}

func firstAdminID(ctx context.Context, db *sqlx.DB) (int64, error) {
	var id int64
	err := db.QueryRowxContext(ctx, `SELECT id FROM users WHERE role = 'admin' AND is_active = true ORDER BY id LIMIT 1`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("no admin user: %w", err)
	}
	return id, nil
}

// =========================
// Modes
// =========================

func runDiscover(ctx context.Context, cfg *importConfig, client *bnovo.Client) error {
	bookings, err := client.ListBookings(ctx, cfg.DateFrom, cfg.DateTo)
	if err != nil {
		return err
	}
	type roomInfo struct {
		BnovoRoomID   int64    `json:"bnovoRoomId"`
		BnovoRoomName string   `json:"bnovoRoomName"`
		Property      string   `json:"property,omitempty"`
		Bookings      int      `json:"bookings"`
		Nights        int      `json:"nights"`
		SampleNumbers []string `json:"sampleNumbers"`
	}
	uniq := map[int64]*roomInfo{}
	for _, b := range bookings {
		// Используем название из price (где оно точно присутствует), фолбэк — top-level.
		name := ""
		for _, p := range b.Prices {
			if p.RoomName != "" {
				name = p.RoomName
				break
			}
		}
		if name == "" {
			name = b.RoomName
		}
		propPrefix := bnovo.ExtractPropertyPrefix(name)
		for _, p := range b.Prices {
			info, ok := uniq[p.RoomID]
			if !ok {
				info = &roomInfo{BnovoRoomID: p.RoomID, BnovoRoomName: name, Property: propPrefix}
				uniq[p.RoomID] = info
			}
			info.Bookings++
			if p.Date != "" {
				info.Nights++
			}
			if len(info.SampleNumbers) < 3 && b.Number != "" {
				info.SampleNumbers = append(info.SampleNumbers, b.Number)
			}
		}
	}
	out := make([]*roomInfo, 0, len(uniq))
	for _, v := range uniq {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Property != out[j].Property {
			return out[i].Property < out[j].Property
		}
		return out[i].BnovoRoomID < out[j].BnovoRoomID
	})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\nВсего броней из Bnovo: %d (за %s — %s)\n", len(bookings), cfg.DateFrom, cfg.DateTo)
	fmt.Fprintf(os.Stderr, "Уникальных bnovo_room_id: %d\n", len(out))
	if len(cfg.PropertyPrefixes) > 0 {
		fmt.Fprintf(os.Stderr, "Фильтр propertyPrefixes: %v\n", cfg.PropertyPrefixes)
	}
	fmt.Fprintln(os.Stderr, "Заполните bnovo-rooms.json: скопируйте bnovoRoomId/bnovoRoomName и проставьте ourRoomId.")
	fmt.Fprintln(os.Stderr, "Используйте поле propertyPrefixes чтобы отсеять лишние объекты (Дмитровское, Смольная…).")
	return nil
}

func runWipe(ctx context.Context, db *sqlx.DB, dryRun bool) error {
	if dryRun {
		var bookings, guests int
		_ = db.QueryRowxContext(ctx, `SELECT count(*) FROM bookings`).Scan(&bookings)
		_ = db.QueryRowxContext(ctx, `SELECT count(*) FROM guests`).Scan(&guests)
		fmt.Printf("[dry-run] будет удалено: bookings=%d guests=%d\n", bookings, guests)
		return nil
	}
	br := repo.NewBookingRepo(db)
	gr := repo.NewGuestRepo(db)
	bk, hist, err := br.WipeAll(ctx)
	if err != nil {
		return fmt.Errorf("wipe bookings: %w", err)
	}
	g, err := gr.WipeAll(ctx)
	if err != nil {
		return fmt.Errorf("wipe guests: %w", err)
	}
	fmt.Printf("wipe: bookings=%d (history rows = %d), guests=%d\n", bk, hist, g)
	return nil
}

func runImport(ctx context.Context, db *sqlx.DB, cfg *importConfig, client *bnovo.Client, dryRun bool, limit int) error {
	bookings, err := client.ListBookings(ctx, cfg.DateFrom, cfg.DateTo)
	if err != nil {
		return err
	}
	if limit > 0 && len(bookings) > limit {
		bookings = bookings[:limit]
	}
	if len(bookings) == 0 {
		fmt.Println("Bnovo не вернул ни одной брони в заданном окне")
		return nil
	}

	// Фильтр по префиксам свойств (если указаны).
	prefixes := cfg.PropertyPrefixes
	if len(prefixes) > 0 {
		before := len(bookings)
		bookings = filterByPrefix(bookings, prefixes)
		fmt.Printf("Фильтр propertyPrefixes=%v: было %d, осталось %d\n", prefixes, before, len(bookings))
	}

	// bnovo_room_id → наш room_id
	mapping := make(map[int64]roomMapping, len(cfg.Rooms))
	for _, r := range cfg.Rooms {
		mapping[r.BnovoRoomID] = r
	}

	createdBy, err := firstAdminID(ctx, db)
	if err != nil {
		return err
	}

	bookingsR := repo.NewBookingRepo(db)
	guestsR := repo.NewGuestRepo(db)
	roomsR := repo.NewRoomRepo(db)
	ratesR := repo.NewRateRepo(db)
	kindsR := repo.NewRoomKindRepo(db)
	bs := service.NewBookingService(bookingsR, guestsR, roomsR, ratesR, kindsR)

	var (
		imported, skippedExists, skippedNoRoom, skippedNoPrices, skippedBadDates, errorsN int
		totalWarnings                                                                      []string
	)

	for _, b := range bookings {
		bnovoID := strconv.FormatInt(b.ID, 10)
		bnovoNumber := b.Number
		fullName := bnovo.FullName(b.Customer)
		phone := strPtr(b.Customer.Phone)
		email := strPtr(b.Customer.Email)
		adults := bnovo.Adults(b)

		// Статус и источник — с детектом «не распознано».
		status, statusOK := bnovo.MapStatus(b.Status.Name)
		source, sourceOK := bnovo.MapSource(b.Source.Name)
		if !statusOK {
			totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: неизвестный статус %q, использую %q", b.ID, b.Status.Name, status))
		}
		if !sourceOK && b.Source.Name != "" {
			totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: неизвестный источник %q, использую %q", b.ID, b.Source.Name, source))
		}

		guest := domain.Guest{FullName: fullName, Phone: phone, Email: email}

		// Даты берём из dates.arrival / dates.departure — это авторитетный источник.
		// Формат: "2026-06-15 14:00:00+03" → берём первые 10 символов.
		if len(b.Dates.Arrival) < 10 || len(b.Dates.Departure) < 10 {
			totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: кривые arrival/departure %q / %q, пропуск", b.ID, b.Dates.Arrival, b.Dates.Departure))
			skippedBadDates++
			continue
		}
		checkIn, err1 := parseDate(b.Dates.Arrival[:10])
		checkOut, err2 := parseDate(b.Dates.Departure[:10])
		if err1 != nil || err2 != nil {
			totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: кривые arrival/departure, пропуск", b.ID))
			skippedBadDates++
			continue
		}
		// Выезд в 12:00, значит checkOut = день выезда как есть (последняя ночь = checkOut-1).
		// Если checkOut <= checkIn — пропуск.
		if !checkOut.After(checkIn) {
			totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: checkOut <= checkIn (%s..%s), пропуск", b.ID, b.Dates.Arrival[:10], b.Dates.Departure[:10]))
			skippedBadDates++
			continue
		}

		// Для каждой уникальной комнаты в prices[] создаём свою бронь.
		roomIDs := bnovo.ExtractRoomIDs(b.Prices)
		if len(roomIDs) == 0 {
			if b.RoomID > 0 {
				roomIDs = []int64{b.RoomID}
			} else {
				totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: prices пуст и room_id=0, пропуск", b.ID))
				skippedNoPrices++
				continue
			}
		}

		// Гость создаётся один раз, и его id используется во всех N бронированиях.
		var guestID *int64
		if !dryRun {
			id, err := guestsR.Create(ctx, guest)
			if err != nil {
				totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d: не удалось создать гостя: %v", b.ID, err))
				errorsN++
				continue
			}
			guestID = &id
		}

		for _, ridBnovo := range roomIDs {
			rmap, ok := mapping[ridBnovo]
			if !ok {
				totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d, bnovo_room_id=%d: нет в маппинге, пропуск комнаты", b.ID, ridBnovo))
				skippedNoRoom++
				continue
			}

			if !dryRun {
				exists, err := bookingsR.ExistsByBnovo(ctx, bnovoID, rmap.OurRoomID)
				if err != nil {
					totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d, room=%d: exists check failed: %v", b.ID, rmap.OurRoomID, err))
					errorsN++
					continue
				}
				if exists {
					skippedExists++
					continue
				}

				// Сумма = сумма цен для этой комнаты в этой брони.
				var total float64
				for _, p := range b.Prices {
					if p.RoomID == ridBnovo {
						total += p.Price
					}
				}
				if total == 0 {
					total = b.Amount // запасной вариант — общая сумма брони
				}

				notes := strPtr(fmt.Sprintf("Bnovo %s (id=%s)", bnovoNumber, bnovoID))
				created, err := bs.Create(ctx, service.CreateBookingInput{
					RoomID:       rmap.OurRoomID,
					CheckIn:      checkIn,
					CheckOut:     checkOut,
					CheckInTime:  "14:00:00",
					CheckOutTime: "12:00:00",
					Adults:       adults,
					Source:       source,
					GuestID:      guestID,
					Total:        &total,
					Prepay:       0,
					Notes:        notes,
					CreatedBy:    createdBy,
				})
				if err != nil {
					totalWarnings = append(totalWarnings, fmt.Sprintf("Bnovo id=%d, room=%d: create failed: %v", b.ID, rmap.OurRoomID, err))
					errorsN++
					continue
				}
				// Прописываем статус (service.Create ставит new; если в Bnovo было иначе — обновим).
				if created.Status != status {
					if err := bookingsR.UpdateStatus(ctx, created.ID, status); err != nil {
						totalWarnings = append(totalWarnings, fmt.Sprintf("booking id=%d: update status to %q failed: %v", created.ID, status, err))
					}
				}
				// Линк с Bnovo.
				if err := bookingsR.UpdateBnovoLink(ctx, created.ID, bnovoID, bnovoNumber); err != nil {
					totalWarnings = append(totalWarnings, fmt.Sprintf("booking id=%d: set bnovo link failed: %v", created.ID, err))
				}
			}
			imported++
			if dryRun {
				fmt.Printf("  [dry] bnovo id=%-9s room=%-6d → our room=%d  %s..%s (%d гостей)\n",
					bnovoID, ridBnovo, rmap.OurRoomID, b.Dates.Arrival[:10], b.Dates.Departure[:10], adults)
			} else {
				fmt.Printf("  bnovo id=%-9s room=%-6d → our room=%d\n", bnovoID, ridBnovo, rmap.OurRoomID)
			}
		}
	}

	if dryRun {
		fmt.Printf("\n[dry-run] было бы создано: %d (skipped exists=%d no-room=%d no-prices=%d bad-dates=%d errors=%d)\n",
			imported, skippedExists, skippedNoRoom, skippedNoPrices, skippedBadDates, errorsN)
	} else {
		fmt.Printf("\nimport: создано=%d (skipped exists=%d no-room=%d no-prices=%d bad-dates=%d errors=%d)\n",
			imported, skippedExists, skippedNoRoom, skippedNoPrices, skippedBadDates, errorsN)
	}
	for _, w := range totalWarnings {
		log.Printf("warn: %s", w)
	}
	return nil
}

// filterByPrefix оставляет только брони, у которых room_name (top-level или из prices[0])
// начинается с одного из префиксов. Чужие объекты (Дмитровское, Смольная) вылетают.
func filterByPrefix(bookings []bnovo.RawBooking, prefixes []string) []bnovo.RawBooking {
	prefixesLower := make([]string, len(prefixes))
	for i, p := range prefixes {
		prefixesLower[i] = strings.ToLower(strings.TrimSpace(p))
	}
	out := make([]bnovo.RawBooking, 0, len(bookings))
	for _, b := range bookings {
		name := b.RoomName
		if name == "" && len(b.Prices) > 0 {
			name = b.Prices[0].RoomName
		}
		prefix := strings.ToLower(bnovo.ExtractPropertyPrefix(name))
		for _, p := range prefixesLower {
			if strings.HasPrefix(prefix, p) {
				out = append(out, b)
				break
			}
		}
	}
	return out
}

// =========================
// main
// =========================

func main() {
	mode := flag.String("mode", "all", "режим: discover|import|wipe|all")
	configPath := flag.String("config", "./bnovo-rooms.json", "путь к конфигу")
	dryRun := flag.Bool("dry-run", false, "только печать, в БД ничего не пишем")
	limit := flag.Int("limit", 0, "максимум броней из Bnovo (0 = все)")
	flag.Parse()

	if err := run(*mode, *configPath, *dryRun, *limit); err != nil {
		log.Fatalf("import-bnovo: %v", err)
	}
}

func run(mode, configPath string, dryRun bool, limit int) error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()

	// Для wipe креды и даты не нужны — берём конфиг упрощённо.
	var impCfg *importConfig
	if mode == "wipe" {
		impCfg, err = loadConfigWipeOnly(configPath)
		if err != nil {
			return err
		}
	} else {
		impCfg, err = loadConfig(configPath)
		if err != nil {
			return err
		}
	}

	client := bnovo.New(bnovo.Config{
		AccountID: impCfg.BnovoAccountID,
		APIKey:    impCfg.BnovoAPIKey,
	})

	// Для discover БД не нужна.
	if mode == "discover" {
		return runDiscover(ctx, impCfg, client)
	}

	db, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	switch mode {
	case "wipe":
		return runWipe(ctx, db, dryRun)
	case "import":
		return runImport(ctx, db, impCfg, client, dryRun, limit)
	case "all":
		if err := runWipe(ctx, db, dryRun); err != nil {
			return err
		}
		return runImport(ctx, db, impCfg, client, dryRun, limit)
	default:
		return fmt.Errorf("неизвестный mode=%q (ожидается discover|import|wipe|all)", mode)
	}
}
