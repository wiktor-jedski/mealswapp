# [ARCH-010] - API Gateway

**Description:** Entry point for all client requests, implementing routing, rate limiting, security header injection, CSRF protection, and request validation.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | RouteHandler, RateLimiter, SecurityHeaderMiddleware, CSRFValidator, RequestValidator, CORSHandler |
| **Dependencies** | All backend services |
| **Traceability** | SW-REQ-064, SW-REQ-067, SW-REQ-068, SW-REQ-076, SW-REQ-078 |

**Dynamic Behavior:**

- **Rate Limiting:** Enforces 10 failed login attempts per IP per 10-minute window. Configurable limits per endpoint.
- **Security Headers:** Injects CSP, X-Frame-Options (DENY), X-Content-Type-Options (nosniff), Referrer-Policy, Permissions-Policy on all responses.
- **CSRF Protection:** Validates synchronizer tokens on all state-changing requests (POST, PUT, DELETE).
- **Timeout Management:** Enforces 10-second timeout on all API requests, returns 504 on timeout.
- **API Versioning:** Routes requests based on version prefix (e.g., /api/v1/) for future mobile integration.

**Interface Definition:**

- `Input`: HTTP requests from clients
- `Output`: Routed requests to services, HTTP responses with security headers

**Alternative Analysis (BP6):**

- *Chosen Approach:* Application-level API gateway (Express/Fastify middleware)
- *Alternative Considered:* Dedicated API gateway service (Kong, AWS API Gateway)
- *Trade-off:* Application-level gateway reduces infrastructure complexity and latency for current scale. Dedicated gateway would provide advanced features (API keys, analytics) but adds operational overhead. For 1000 concurrent users (SW-REQ-082), application-level gateway is sufficient and simpler to deploy.

**Reference Documentation:** 
- 02_APPENDIX_A.md
