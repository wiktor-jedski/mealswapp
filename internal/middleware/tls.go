// Phase: phase-01 | Task: 12 | Architecture: ARCH-013 | Design: TLSEnforcer

package middleware

import (
	"crypto/tls"
	"errors"

	"github.com/gofiber/fiber/v2"
)

type TLSConfig struct {
	Enabled      bool
	MinVersion   uint16
	RedirectHTTP bool
	RedirectCode int
}

type EnforcerConfig struct {
	TLSConfig    TLSConfig
	Next         func(*fiber.Ctx) bool
	ErrorHandler func(*fiber.Ctx, error) error
}

var (
	ErrConnectionClosed = errors.New("tls: client terminated connection during handshake")
	ErrInvalidProtocol  = errors.New("tls: handshake failure due to unsupported TLS version")
	ErrMissingHost      = errors.New("tls: cannot construct redirect URL - missing host header")
)

func NewTLSEnforcer(config EnforcerConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if config.Next != nil && config.Next(c) {
			return c.Next()
		}

		if config.TLSConfig.RedirectHTTP && c.Protocol() == "http" {
			host := c.Hostname()
			if host == "" {
				if config.ErrorHandler != nil {
					return config.ErrorHandler(c, ErrMissingHost)
				}
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": ErrMissingHost.Error(),
				})
			}

			redirectCode := config.TLSConfig.RedirectCode
			if redirectCode == 0 {
				redirectCode = fiber.StatusMovedPermanently
			}

			httpsURL := "https://" + host + c.Path()
			if queryStr := string(c.Request().URI().QueryString()); queryStr != "" {
				httpsURL += "?" + queryStr
			}

			return c.Redirect(httpsURL, redirectCode)
		}

		return c.Next()
	}
}

func InitializeTLS(app *fiber.App, certFile, keyFile string) error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	app.Server().TLSConfig = tlsConfig

	return app.ListenTLS(":443", certFile, keyFile)
}

func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:      true,
		MinVersion:   tls.VersionTLS13,
		RedirectHTTP: true,
		RedirectCode: fiber.StatusMovedPermanently,
	}
}

func DefaultEnforcerConfig() EnforcerConfig {
	return EnforcerConfig{
		TLSConfig: DefaultTLSConfig(),
	}
}
