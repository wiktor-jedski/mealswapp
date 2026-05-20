package session

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	AccessCookieName  = "access_token"
	RefreshCookieName = "refresh_token"
)

type Config struct {
	Environment string
	Domain      string
}

type Manager struct {
	config Config
}

func NewManager(config Config) Manager {
	return Manager{config: config}
}

func (manager Manager) SetAuthCookies(ctx *fiber.Ctx, accessToken string, refreshToken string, accessExpiresAt time.Time, refreshExpiresAt time.Time) {
	ctx.Cookie(manager.cookie(AccessCookieName, accessToken, accessExpiresAt))
	ctx.Cookie(manager.cookie(RefreshCookieName, refreshToken, refreshExpiresAt))
}

func (manager Manager) ClearAuthCookies(ctx *fiber.Ctx) {
	ctx.Cookie(manager.cookie(AccessCookieName, "", time.Now().Add(-time.Hour)))
	ctx.Cookie(manager.cookie(RefreshCookieName, "", time.Now().Add(-time.Hour)))
}

func (manager Manager) cookie(name string, value string, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		HTTPOnly: true,
		Secure:   manager.secureCookies(),
		SameSite: manager.sameSite(),
		Domain:   manager.config.Domain,
		Path:     "/",
	}
}

func (manager Manager) secureCookies() bool {
	return manager.config.Environment != "" && manager.config.Environment != "local" && manager.config.Environment != "test"
}

func (manager Manager) sameSite() string {
	if manager.secureCookies() {
		return "Strict"
	}
	return "Lax"
}
