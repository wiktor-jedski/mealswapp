## Phase 6: Subscriptions & User Profile

**Goal:** Implement payment processing and user data management

### Components & Static Aspects

#### ARCH-007 - Subscription Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **SubscriptionController** | HTTP handlers for subscription endpoints | `subscription/controller.go` |
| **StripeWebhookHandler** | Process payment_intent.succeeded/failed events | `subscription/webhook_handler.go` |
| **EntitlementManager** | Check/update user entitlement status | `subscription/entitlement_manager.go` |
| **TrialTracker** | 7-day trial activation and expiration | `subscription/trial_tracker.go` |
| **UsageLimiter** | 3 searches/24h for free tier | `subscription/usage_limiter.go` |

#### ARCH-008 - User Profile Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ProfileController** | HTTP handlers for profile endpoints | `profile/controller.go` |
| **PreferenceManager** | Unit preferences, theme persistence | `profile/preference_manager.go` |
| **SavedDataRepository** | Saved items, diets, favorites | `profile/saved_data_repo.go` |
| **DataExporter** | Export user data to JSON/CSV | `profile/exporter.go` |
| **AccountDeleter** | Cascade delete all user data | `profile/deleter.go` |

#### ARCH-015 - Compliance Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ConsentManager** | Track Privacy Policy/ToS consent | `compliance/consent_manager.go` |
| **DisclaimerRenderer** | Medical disclaimer content | `compliance/disclaimer_renderer.go` |
| **DataRetentionPolicy** | 30-day backup retention rules | `compliance/retention_policy.go` |
| **BackupManager** | Daily backup coordination | `compliance/backup_manager.go` |

### Testing
- [ ] Stripe Elements tokenization (no raw card data on server)
- [ ] Webhook idempotency (duplicate events ignored)
- [ ] Webhook signature verification (reject invalid signatures)
- [ ] Free tier enforces 3 searches/24h
- [ ] Paid features blocked for free users
- [ ] 7-day trial activates on first OAuth
- [ ] Trial auto-downgrades after 7 days
- [ ] Data export includes all PII, saved items, history
- [ ] Account deletion cascades to all related data
- [ ] Consent checkbox required for registration
- [ ] Medical disclaimer displayed on login

---

