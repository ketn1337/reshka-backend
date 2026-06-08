package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ketn1337/reshka-backend/internal/auth"
	"github.com/ketn1337/reshka-backend/internal/domain"
)

const cookieName = "reshka_token"

// CookieName возвращает имя cookie, чтобы logout-handler мог его стереть.
func CookieName() string { return cookieName }

// SetAuthCookie ставит httpOnly-cookie с токеном.
func SetAuthCookie(c *gin.Context, token string, ttlSeconds int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, token, ttlSeconds, "/", "", false, true)
}

// ClearAuthCookie стирает cookie.
func ClearAuthCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, "", -1, "/", "", false, true)
}

func tokenFromRequest(c *gin.Context) string {
	if t, err := c.Cookie(cookieName); err == nil && t != "" {
		return t
	}
	// Фоллбэк для API-клиентов без cookie: Authorization: Bearer ...
	h := c.GetHeader("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}

// RequireAuth парсит JWT и кладёт claims в контекст. 401 если невалидный.
func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := tokenFromRequest(c)
		if tok == "" {
			abortErr(c, http.StatusUnauthorized, "unauthorized", "missing token")
			return
		}
		claims, err := auth.Parse(secret, tok)
		if err != nil {
			abortErr(c, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}
		auth.SetClaims(c, claims.UserID, claims.Role, claims.Email)
		c.Next()
	}
}

// RequireRole проверяет, что роль пользователя входит в список.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := auth.RoleFromCtx(c)
		if !ok {
			abortErr(c, http.StatusUnauthorized, "unauthorized", "no role")
			return
		}
		if _, ok := allowed[role]; !ok {
			abortErr(c, http.StatusForbidden, "forbidden", "role not allowed")
			return
		}
		c.Next()
	}
}

func abortErr(c *gin.Context, code int, errCode, msg string) {
	c.AbortWithStatusJSON(code, gin.H{
		"error": gin.H{
			"code":    errCode,
			"message": msg,
		},
	})
}

// MapDomainError переводит доменную ошибку в HTTP. Используется в handler.
func MapDomainError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		abortErr(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrConflict):
		abortErr(c, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		abortErr(c, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		abortErr(c, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, domain.ErrValidation):
		abortErr(c, http.StatusUnprocessableEntity, "validation", err.Error())
	case errors.Is(err, domain.ErrBadStatus):
		abortErr(c, http.StatusConflict, "bad_status", err.Error())
	default:
		abortErr(c, http.StatusInternalServerError, "internal", err.Error())
	}
	return true
}
