package auth

import "github.com/gin-gonic/gin"

// Context-ключи.
const (
	CtxUserID = "userID"
	CtxRole   = "userRole"
	CtxEmail  = "userEmail"
)

// SetClaims кладёт userID/role/email в gin.Context.
func SetClaims(c *gin.Context, userID int64, role, email string) {
	c.Set(CtxUserID, userID)
	c.Set(CtxRole, role)
	c.Set(CtxEmail, email)
}

// UserIDFromCtx возвращает userID и флаг наличия.
func UserIDFromCtx(c *gin.Context) (int64, bool) {
	v, ok := c.Get(CtxUserID)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// RoleFromCtx возвращает роль.
func RoleFromCtx(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxRole)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// EmailFromCtx возвращает email.
func EmailFromCtx(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxEmail)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
