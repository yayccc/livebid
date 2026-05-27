package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingSecret    = errors.New("auth: missing jwt secret")
	ErrInvalidClaims    = errors.New("auth: invalid jwt claims")
	ErrInvalidToken     = errors.New("auth: invalid jwt token")
	ErrInvalidSignature = errors.New("auth: invalid jwt signature")
	ErrTokenExpired     = errors.New("auth: jwt token expired")
)

type Claims struct {
	Issuer    string    // 签发方 例如 livebid
	Subject   string    // 主体 ID，用户 ID、商铺 ID 或管理员 ID
	ID        string    // token 唯一 ID，对应 JWT jti，可选
	ExpiresAt time.Time // 过期时间
}

type JWTManager struct {
	secret []byte
	issuer string
	now    func() time.Time
}

type JWTOption func(*JWTManager)

func WithIssuer(issuer string) JWTOption {
	return func(m *JWTManager) {
		m.issuer = issuer
	}
}

func WithNow(now func() time.Time) JWTOption {
	return func(m *JWTManager) {
		if now != nil {
			m.now = now
		}
	}
}

func NewJWTManager(secret string, opts ...JWTOption) (*JWTManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrMissingSecret
	}

	manager := &JWTManager{
		secret: []byte(secret),
		now:    time.Now,
	}
	for _, opt := range opts {
		opt(manager)
	}
	return manager, nil
}

func (m *JWTManager) Sign(claims Claims) (string, error) {
	if m == nil || len(m.secret) == 0 {
		return "", ErrMissingSecret
	}
	if strings.TrimSpace(claims.Subject) == "" || claims.ExpiresAt.IsZero() {
		return "", ErrInvalidClaims
	}

	now := m.now().UTC()
	if claims.Issuer == "" {
		claims.Issuer = m.issuer
	}
	if !claims.ExpiresAt.After(now) {
		return "", ErrInvalidClaims
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims.toJWTClaims())
	return token.SignedString(m.secret)
}

func (m *JWTManager) Verify(token string) (Claims, error) {
	if m == nil || len(m.secret) == 0 {
		return Claims{}, ErrMissingSecret
	}

	if strings.TrimSpace(token) == "" {
		return Claims{}, ErrInvalidToken
	}

	options := []jwt.ParserOption{
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time {
			return m.now().UTC()
		}),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	}
	if m.issuer != "" {
		options = append(options, jwt.WithIssuer(m.issuer))
	}

	parsedClaims := &jwtClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, parsedClaims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	}, options...)
	if err != nil {
		return Claims{}, mapJWTError(err)
	}

	if parsedToken == nil || !parsedToken.Valid || strings.TrimSpace(parsedClaims.Subject) == "" {
		return Claims{}, ErrInvalidToken
	}

	return parsedClaims.toClaims(), nil
}

type jwtClaims struct {
	jwt.RegisteredClaims
}

func (c Claims) toJWTClaims() jwtClaims {
	return jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.Issuer,
			Subject:   c.Subject,
			ID:        c.ID,
			ExpiresAt: jwt.NewNumericDate(c.ExpiresAt.UTC()),
		},
	}
}

func (c jwtClaims) toClaims() Claims {
	return Claims{
		Issuer:    c.Issuer,
		Subject:   c.Subject,
		ID:        c.ID,
		ExpiresAt: numericDateTime(c.ExpiresAt),
	}
}

func numericDateTime(date *jwt.NumericDate) time.Time {
	if date == nil {
		return time.Time{}
	}
	return date.Time.UTC()
}

func mapJWTError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return ErrInvalidSignature
	case errors.Is(err, jwt.ErrTokenMalformed),
		errors.Is(err, jwt.ErrTokenUnverifiable),
		errors.Is(err, jwt.ErrTokenRequiredClaimMissing),
		errors.Is(err, jwt.ErrTokenInvalidIssuer),
		errors.Is(err, jwt.ErrTokenInvalidClaims):
		return ErrInvalidToken
	default:
		return ErrInvalidToken
	}
}
