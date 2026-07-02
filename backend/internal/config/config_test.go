package config

// Implements DESIGN-010 RequestValidator configuration verification.

import (
	"testing"
	"time"
)

func setProductionConfig(t *testing.T) {
	t.Helper()
	t.Setenv("MEALSWAPP_ENV", "production")
	t.Setenv("MEALSWAPP_DATABASE_URL", "postgres://example")
	t.Setenv("MEALSWAPP_REDIS_URL", "redis://example:6379/0")
	t.Setenv("MEALSWAPP_FRONTEND_ORIGIN", "https://example.test")
	t.Setenv("MEALSWAPP_ALLOWED_ORIGINS", "https://example.test")
	t.Setenv("MEALSWAPP_PRIVACY_POLICY_VERSION", "privacy-2026-06")
	t.Setenv("MEALSWAPP_TERMS_VERSION", "terms-2026-06")
	t.Setenv("MEALSWAPP_STRIPE_SECRET_KEY", "sk_live_fixture")
	t.Setenv("MEALSWAPP_STRIPE_WEBHOOK_SECRET", "whsec_live_fixture")
	t.Setenv("MEALSWAPP_STRIPE_MONTHLY_PRICE_ID", "price_live_monthly_fixture")
	t.Setenv("MEALSWAPP_STRIPE_ANNUAL_PRICE_ID", "price_live_annual_fixture")
	t.Setenv("MEALSWAPP_CHECKOUT_SUCCESS_URL", "https://example.test/billing/success")
	t.Setenv("MEALSWAPP_CHECKOUT_CANCEL_URL", "https://example.test/billing/cancel")
}

// TestLoadUsesDevelopmentDefaults proves that config will use default values
// if no other values are passed.
// TestLoadUsesDevelopmentDefaults verifies DESIGN-010 RequestValidator config defaults.
func TestLoadUsesDevelopmentDefaults(t *testing.T) {
	t.Setenv("MEALSWAPP_HTTP_PORT", "")
	t.Setenv("MEALSWAPP_DATABASE_URL", "")
	t.Setenv("MEALSWAPP_REDIS_URL", "")
	t.Setenv("MEALSWAPP_ENV", "")
	t.Setenv("MEALSWAPP_FRONTEND_ORIGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPPort != defaultHTTPPort {
		t.Fatalf("HTTPPort = %q, want %q", cfg.HTTPPort, defaultHTTPPort)
	}
	if cfg.Environment != defaultEnvironment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, defaultEnvironment)
	}
	if cfg.HSTSMaxAge != defaultHSTSMaxAge {
		t.Fatalf("HSTSMaxAge = %d, want %d", cfg.HSTSMaxAge, defaultHSTSMaxAge)
	}
	if cfg.Account.PasswordMinLength != 12 || cfg.Account.AccessTokenTTL != 15*time.Minute || cfg.Account.RefreshTokenTTL != 7*24*time.Hour {
		t.Fatalf("unexpected account defaults: %+v", cfg.Account)
	}
	if cfg.Account.AccessCookieName != "mealswapp_access" || cfg.Account.RefreshCookieName != "mealswapp_refresh" {
		t.Fatalf("unexpected cookie names: %+v", cfg.Account)
	}
	if cfg.Billing.StripeSecretKey != defaultStripeSecretKey || cfg.Billing.StripeWebhookSecret != defaultStripeWebhookSecret {
		t.Fatalf("unexpected billing defaults: %+v", cfg.Billing)
	}
}

// TestLoadRequiresProductionDependencyURLs proves that config will not load
// if prod env lacks valid URLs.
// TestLoadRequiresProductionDependencyURLs verifies DESIGN-010 RequestValidator production guards.
func TestLoadRequiresProductionDependencyURLs(t *testing.T) {
	t.Setenv("MEALSWAPP_ENV", "production")
	t.Setenv("MEALSWAPP_DATABASE_URL", "")
	t.Setenv("MEALSWAPP_REDIS_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want production dependency URL error")
	}
}

// TestLoadAcceptsAccountOverrides verifies DESIGN-006 AuthController account-flow configuration.
func TestLoadAcceptsAccountOverrides(t *testing.T) {
	t.Setenv("MEALSWAPP_PASSWORD_MIN_LENGTH", "14")
	t.Setenv("MEALSWAPP_ACCESS_TOKEN_TTL", "20m")
	t.Setenv("MEALSWAPP_REFRESH_TOKEN_TTL", "240h")
	t.Setenv("MEALSWAPP_ACCESS_COOKIE_NAME", "__Host-custom_access")
	t.Setenv("MEALSWAPP_REFRESH_COOKIE_NAME", "__Host-custom_refresh")
	t.Setenv("MEALSWAPP_PRIVACY_POLICY_VERSION", "privacy-2026-06")
	t.Setenv("MEALSWAPP_TERMS_VERSION", "terms-2026-06")
	t.Setenv("MEALSWAPP_DISCLAIMER_FALLBACK_VERSION", "medical-2026-06")
	t.Setenv("MEALSWAPP_EMAIL_VERIFICATION_TTL", "48h")
	t.Setenv("MEALSWAPP_PASSWORD_RESET_TTL", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Account.PasswordMinLength != 14 || cfg.Account.AccessTokenTTL != 20*time.Minute || cfg.Account.RefreshTokenTTL != 240*time.Hour {
		t.Fatalf("unexpected account override: %+v", cfg.Account)
	}
	if cfg.Account.EmailVerificationTTL != 48*time.Hour || cfg.Account.PasswordResetTTL != 30*time.Minute {
		t.Fatalf("unexpected account expiry override: %+v", cfg.Account)
	}
}

// TestLoadRejectsInvalidAccountSettings verifies DESIGN-006 AuthController account-flow guards.
func TestLoadRejectsInvalidAccountSettings(t *testing.T) {
	for key, value := range map[string]string{
		"MEALSWAPP_PASSWORD_MIN_LENGTH":         "7",
		"MEALSWAPP_ACCESS_TOKEN_TTL":            "bad",
		"MEALSWAPP_REFRESH_TOKEN_TTL":           "1m",
		"MEALSWAPP_ACCESS_COOKIE_NAME":          " ",
		"MEALSWAPP_REFRESH_COOKIE_NAME":         " ",
		"MEALSWAPP_PRIVACY_POLICY_VERSION":      " ",
		"MEALSWAPP_TERMS_VERSION":               " ",
		"MEALSWAPP_DISCLAIMER_FALLBACK_VERSION": " ",
		"MEALSWAPP_EMAIL_VERIFICATION_TTL":      "0s",
		"MEALSWAPP_PASSWORD_RESET_TTL":          "-1s",
	} {
		t.Run(key, func(t *testing.T) {
			if key == "MEALSWAPP_REFRESH_COOKIE_NAME" {
				t.Setenv("MEALSWAPP_ACCESS_COOKIE_NAME", "same")
				t.Setenv("MEALSWAPP_REFRESH_COOKIE_NAME", "same")
			} else {
				t.Setenv(key, value)
			}
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted %s=%q", key, value)
			}
		})
	}
}

func TestLoadRejectsMalformedRefreshTTL(t *testing.T) {
	t.Setenv("MEALSWAPP_REFRESH_TOKEN_TTL", "bad")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted malformed refresh TTL")
	}
}

// TestLoadRejectsDevelopmentAccountSettingsInProduction verifies DESIGN-006 AuthController production validation.
func TestLoadRejectsDevelopmentAccountSettingsInProduction(t *testing.T) {
	t.Run("default consent version", func(t *testing.T) {
		setProductionConfig(t)
		t.Setenv("MEALSWAPP_PRIVACY_POLICY_VERSION", "")
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted development privacy version in production")
		}
	})
	t.Run("development cookie name", func(t *testing.T) {
		setProductionConfig(t)
		t.Setenv("MEALSWAPP_ACCESS_COOKIE_NAME", "dev_access")
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted development cookie name in production")
		}
	})
}

// TestLoadAcceptsProductionDependencyURLs proves that config app will load
// if all necessary URLs are passed.
// TestLoadAcceptsProductionDependencyURLs verifies DESIGN-010 RequestValidator production overrides.
func TestLoadAcceptsProductionDependencyURLs(t *testing.T) {
	setProductionConfig(t)
	t.Setenv("MEALSWAPP_HTTP_PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPPort != "9090" || cfg.FrontendOrigin != "https://example.test" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

// TestLoadRejectsInvalidRedisURL checks if the string is indeed a valid redis URL.
// TestLoadRejectsInvalidRedisURL verifies DESIGN-010 RequestValidator Redis URL validation.
func TestLoadRejectsInvalidRedisURL(t *testing.T) {
	t.Setenv("MEALSWAPP_REDIS_URL", "not a redis url")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want Redis URL validation error")
	}
}

// TestLoadRejectsInvalidDatabaseURL checks if the string is indeed a db URL.
// TestLoadRejectsInvalidDatabaseURL verifies DESIGN-010 RequestValidator database URL validation.
func TestLoadRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("MEALSWAPP_DATABASE_URL", "https://example.test")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want database URL validation error")
	}
}

// TestLoadRejectsInvalidFrontendOrigin verifies DESIGN-010 RequestValidator frontend origin validation.
func TestLoadRejectsInvalidFrontendOrigin(t *testing.T) {
	t.Setenv("MEALSWAPP_FRONTEND_ORIGIN", "not a frontend origin")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want frontend origin validation error")
	}
}

// TestLoadRejectsInvalidGatewaySettings verifies DESIGN-010 RequestValidator gateway value validation.
func TestLoadRejectsInvalidGatewaySettings(t *testing.T) {
	for key, value := range map[string]string{
		"MEALSWAPP_API_TIMEOUT":     "bad",
		"MEALSWAPP_TRUST_PROXY":     "bad",
		"MEALSWAPP_ENFORCE_TLS":     "bad",
		"MEALSWAPP_HSTS_MAX_AGE":    "-1",
		"MEALSWAPP_ALLOWED_ORIGINS": "bad",
		"MEALSWAPP_TLS_MIN_VERSION": "1.2",
	} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted %s=%q", key, value)
			}
		})
	}
}

// TestLoadRejectsEmptyAllowedOrigins verifies DESIGN-010 CORSHandler origin-list validation.
func TestLoadRejectsEmptyAllowedOrigins(t *testing.T) {
	t.Setenv("MEALSWAPP_ALLOWED_ORIGINS", ", ")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an empty allowed-origin list")
	}
}

// TestLoadRejectsTrustedProxyUntilIngressExists verifies DESIGN-013 TLSEnforcer deployment deferral.
func TestLoadRejectsTrustedProxyUntilIngressExists(t *testing.T) {
	t.Setenv("MEALSWAPP_TRUST_PROXY", "true")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted trusted-proxy mode before Phase 09 ingress enforcement")
	}
}

// TestLoadAcceptsHSTSMaxAgeOverride verifies DESIGN-010 SecurityHeaderMiddleware HSTS configuration.
func TestLoadAcceptsHSTSMaxAgeOverride(t *testing.T) {
	t.Setenv("MEALSWAPP_HSTS_MAX_AGE", "0")
	cfg, err := Load()
	if err != nil || cfg.HSTSMaxAge != 0 {
		t.Fatalf("Load() = %+v, %v", cfg, err)
	}
}

// TestLoadAcceptsMultipleOrigins verifies DESIGN-010 CORSHandler origin-list parsing.
func TestLoadAcceptsMultipleOrigins(t *testing.T) {
	t.Setenv("MEALSWAPP_ALLOWED_ORIGINS", "https://one.test, https://two.test")
	t.Setenv("MEALSWAPP_CHECKOUT_SUCCESS_URL", "https://one.test/billing/success")
	t.Setenv("MEALSWAPP_CHECKOUT_CANCEL_URL", "https://two.test/billing/cancel")
	cfg, err := Load()
	if err != nil || len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("Load() = %+v, %v", cfg, err)
	}
}

// TestLoadAcceptsStripeSandboxFixtures verifies DESIGN-007 SubscriptionController sandbox configuration loading.
func TestLoadAcceptsStripeSandboxFixtures(t *testing.T) {
	t.Setenv("MEALSWAPP_STRIPE_SECRET_KEY", "sk_test_sandbox_fixture")
	t.Setenv("MEALSWAPP_STRIPE_WEBHOOK_SECRET", "whsec_sandbox_fixture")
	t.Setenv("MEALSWAPP_STRIPE_MONTHLY_PRICE_ID", "price_sandbox_monthly_fixture")
	t.Setenv("MEALSWAPP_STRIPE_ANNUAL_PRICE_ID", "price_sandbox_annual_fixture")
	t.Setenv("MEALSWAPP_CHECKOUT_SUCCESS_URL", "http://localhost:5173/checkout/success")
	t.Setenv("MEALSWAPP_CHECKOUT_CANCEL_URL", "http://localhost:5173/checkout/cancel")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Billing.StripeSecretKey != "sk_test_sandbox_fixture" || cfg.Billing.StripeWebhookSecret != "whsec_sandbox_fixture" {
		t.Fatalf("unexpected sandbox billing config: %+v", cfg.Billing)
	}
}

// TestLoadMapsSubscriptionPlanPrices verifies DESIGN-007 SubscriptionController and SW-REQ-050 mapping.
func TestLoadMapsSubscriptionPlanPrices(t *testing.T) {
	t.Setenv("MEALSWAPP_STRIPE_MONTHLY_PRICE_ID", "price_monthly_fixture")
	t.Setenv("MEALSWAPP_STRIPE_ANNUAL_PRICE_ID", "price_annual_fixture")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Billing.MonthlyPlan.Code != "monthly" || cfg.Billing.MonthlyPlan.Label != "Monthly" || cfg.Billing.MonthlyPlan.AmountCents != 300 || cfg.Billing.MonthlyPlan.PriceID != "price_monthly_fixture" {
		t.Fatalf("monthly plan = %+v", cfg.Billing.MonthlyPlan)
	}
	if cfg.Billing.AnnualPlan.Code != "annual" || cfg.Billing.AnnualPlan.Label != "Annual" || cfg.Billing.AnnualPlan.AmountCents != 2500 || cfg.Billing.AnnualPlan.PriceID != "price_annual_fixture" {
		t.Fatalf("annual plan = %+v", cfg.Billing.AnnualPlan)
	}
}

// TestLoadRejectsInvalidBillingSettings verifies DESIGN-007 SubscriptionController fail-closed config guards.
func TestLoadRejectsInvalidBillingSettings(t *testing.T) {
	for key, value := range map[string]string{
		"MEALSWAPP_STRIPE_SECRET_KEY":       "pk_test_not_secret",
		"MEALSWAPP_STRIPE_WEBHOOK_SECRET":   "secret",
		"MEALSWAPP_STRIPE_MONTHLY_PRICE_ID": "prod_monthly",
		"MEALSWAPP_STRIPE_ANNUAL_PRICE_ID":  "prod_annual",
		"MEALSWAPP_CHECKOUT_SUCCESS_URL":    "https://evil.test/success",
		"MEALSWAPP_CHECKOUT_CANCEL_URL":     "http://localhost:5173/cancel#fragment",
	} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted %s=%q", key, value)
			}
		})
	}
}

// TestLoadRequiresProductionStripeValues verifies DESIGN-007 SubscriptionController production Stripe guards.
func TestLoadRequiresProductionStripeValues(t *testing.T) {
	for name, mutate := range map[string]func(*testing.T){
		"missing key": func(t *testing.T) {
			t.Setenv("MEALSWAPP_STRIPE_SECRET_KEY", "")
		},
		"test key": func(t *testing.T) {
			t.Setenv("MEALSWAPP_STRIPE_SECRET_KEY", "sk_test_sandbox_fixture")
		},
		"default webhook secret": func(t *testing.T) {
			t.Setenv("MEALSWAPP_STRIPE_WEBHOOK_SECRET", defaultStripeWebhookSecret)
		},
		"default price id": func(t *testing.T) {
			t.Setenv("MEALSWAPP_STRIPE_MONTHLY_PRICE_ID", defaultStripeMonthlyPriceID)
		},
		"insecure redirect": func(t *testing.T) {
			t.Setenv("MEALSWAPP_CHECKOUT_SUCCESS_URL", "http://example.test/billing/success")
		},
	} {
		t.Run(name, func(t *testing.T) {
			setProductionConfig(t)
			mutate(t)
			if _, err := Load(); err == nil {
				t.Fatal("Load() accepted invalid production billing config")
			}
		})
	}
}
