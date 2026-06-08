package bnovo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient — абстракция, чтобы тесты могли подменять транспорт.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

const defaultBaseURL = "https://api.pms.bnovo.ru"
const pageSize = 50 // Bnovo ограничивает 1..50

// Client — тонкая обёртка над REST API Bnovo с кэшем токена.
type Client struct {
	cfg   Config
	http  HTTPClient
	token cachedToken
}

// New создаёт клиент. Если cfg.BaseURL пустой — берётся продовый.
func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}
}

// Auth получает (или возвращает из кэша) access_token.
// POST /api/v1/auth с {id, password}. Кэшируется до exp - 60s.
func (c *Client) Auth(ctx context.Context) (string, error) {
	now := time.Now()
	if c.token.accessToken != "" && c.token.expiresAt.After(now.Add(60*time.Second)) {
		return c.token.accessToken, nil
	}
	body, _ := json.Marshal(authRequest{ID: c.cfg.AccountID, Password: c.cfg.APIKey})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/auth", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth: status %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var ar authResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return "", fmt.Errorf("auth: decode: %w", err)
	}
	// Bnovo оборачивает ответ в {"data": {...}} — разворачиваем если есть.
	if ar.AccessToken == "" && ar.Data != nil {
		ar = *ar.Data
	}
	if ar.AccessToken == "" {
		return "", fmt.Errorf("auth: empty access_token in response: %s", truncate(string(raw), 200))
	}
	ttl := time.Duration(ar.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	c.token = cachedToken{accessToken: ar.AccessToken, expiresAt: time.Now().Add(ttl)}
	return ar.AccessToken, nil
}

// ListBookings тянет ВСЕ брони в [from, to] с пагинацией по 50.
// Bnovo интерпретирует date_from/date_to по дате заезда (arrival) и
// отдаёт все статусы (см. README bnovo_api).
func (c *Client) ListBookings(ctx context.Context, from, to string) ([]RawBooking, error) {
	token, err := c.Auth(ctx)
	if err != nil {
		return nil, err
	}

	var all []RawBooking
	offset := 0
	for {
		page, err := c.fetchBookingsPage(ctx, token, from, to, offset, pageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		// Стоп, если страница неполная или массив пуст.
		if len(page) < pageSize {
			return all, nil
		}
		offset += pageSize
	}
}

func (c *Client) fetchBookingsPage(ctx context.Context, token, from, to string, offset, limit int) ([]RawBooking, error) {
	q := url.Values{}
	q.Set("date_from", from)
	q.Set("date_to", to)
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	u := c.cfg.BaseURL + "/api/v1/bookings?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bookings: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	// Bnovo возвращает 401 при протухшем токене → обновим и повторим один раз.
	if resp.StatusCode == http.StatusUnauthorized {
		c.token = cachedToken{} // инвалидируем
		token, err = c.Auth(ctx)
		if err != nil {
			return nil, err
		}
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req2.Header.Set("Accept", "application/json")
		req2.Header.Set("Authorization", "Bearer "+token)
		resp2, err := c.http.Do(req2)
		if err != nil {
			return nil, fmt.Errorf("bookings retry: %w", err)
		}
		defer resp2.Body.Close()
		raw, _ = io.ReadAll(resp2.Body)
		resp = resp2
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bookings: status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	// Bnovo в нашем аккаунте отдаёт {"data":{"bookings":[...],"meta":{...}},"meta":{...}}.
	// На всякий случай поддерживаем и плоский массив (если когда-то изменится).
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var arr []RawBooking
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return nil, fmt.Errorf("bookings: decode array: %w", err)
		}
		return arr, nil
	}
	var wrapped BookingsListResponse
	if err := json.Unmarshal(trimmed, &wrapped); err != nil {
		return nil, fmt.Errorf("bookings: decode object: %w", err)
	}
	if wrapped.Data != nil {
		return wrapped.Data.Bookings, nil
	}
	return nil, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
