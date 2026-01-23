# TLSEnforcer

**Traceability:** ARCH-013

## 1. Data Structures & Types

```go
package middleware

type TLSConfig struct {
    Enabled bool
    MinVersion uint16
    RedirectHTTP bool
    RedirectCode int
}

type EnforcerConfig struct {
    TLSConfig TLSConfig
    Next func(*fiber.Ctx) bool
    ErrorHandler func(*fiber.Ctx, error) error
}
```

## 2. Logic & Algorithms

1. **Initialization**
   - Read TLS configuration from environment variables or config file.
   - Set `TLSConfig.MinVersion` to `tls.VersionTLS13`.
   - Configure redirect behavior: if `RedirectHTTP` is true, set `RedirectCode` (default 301).

2. **HTTP to HTTPS Redirect Middleware**
   - Check if `RedirectHTTP` is enabled.
   - If the incoming request is HTTP (detected via `ctx.Protocol() == "http"`):
     - Construct the HTTPS URL using the same host and path.
     - Return a redirect response with the configured `RedirectCode`.
     - Stop further middleware processing.

3. **TLS Version Enforcement**
   - When the application is configured to listen on HTTPS directly:
     - The Fiber application must be initialized with a TLS certificate and key.
     - The underlying `net/http` server will enforce the `MinVersion` configured in `TLSConfig`.
     - If a client attempts to connect with a lower TLS version, the connection is terminated by the network layer.

4. **Request Processing**
   - Continue to the next middleware or handler if:
     - The request is already over HTTPS.
     - The request is HTTP but redirection is disabled (not recommended for production).

## 3. State Management & Error Handling

- **Error States:**
  - `ErrConnectionClosed`: Client terminated the connection during handshake or redirect processing.
  - `ErrInvalidProtocol`: Handshake failure due to unsupported TLS version (handled by `net/http`).

- **State Transitions:**
  - `HTTP Request` -> `Redirect to HTTPS` -> `HTTPS Request` -> `Process Request`.
  - `HTTPS Request` -> `Process Request`.

- **Error Handling:**
  - If TLS configuration is invalid (e.g., missing certificate paths), the application will fail to start.
  - If `RedirectHTTP` is enabled and the host header is missing, the redirect will fail to construct a valid URL.

## 4. Component Interfaces

```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
)

func NewTLSEnforcer(config EnforcerConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        if config.Next != nil && config.Next(c) {
            return c.Next()
        }

        if config.TLSConfig.RedirectHTTP && c.Protocol() == "http" {
            httpsHost := c.Hostname() + c.Path()
            return c.Redirect("https://"+httpsHost, config.TLSConfig.RedirectCode)
        }

        return c.Next()
    }
}

func InitializeTLS(app *fiber.App, certFile, keyFile string) error {
    return app.Listener(":443", fiber.ListenConfig{
        TLSConfig: &tls.Config{
            MinVersion: tls.VersionTLS13,
        },
    })
}
```
