package identity

import "github.com/gin-gonic/gin"

type Kind string

const (
	KindShop Kind = "shop"
	KindUser Kind = "user"
)

const principalKey = "identity_principal"

type Principal struct {
	Kind    Kind
	ID      int64
	Subject string
}

func Set(c *gin.Context, principal Principal) {
	c.Set(principalKey, principal)
}

func Get(c *gin.Context) (Principal, bool) {
	value, ok := c.Get(principalKey)
	if !ok {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok && principal.Valid()
}

func ShopID(c *gin.Context) (int64, bool) {
	principal, ok := Get(c)
	if !ok || principal.Kind != KindShop {
		return 0, false
	}
	return principal.ID, true
}

func UserID(c *gin.Context) (int64, bool) {
	principal, ok := Get(c)
	if !ok || principal.Kind != KindUser {
		return 0, false
	}
	return principal.ID, true
}

func (p Principal) Valid() bool {
	return p.Kind != "" && p.ID > 0 && p.Subject != ""
}
