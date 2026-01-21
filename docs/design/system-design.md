# System Design

## Tech Stack

### Frontend (Web)
SvelteKit is the web framework
TypeScript is used throughout
Tailwind CSS handles styling
Skeleton UI or shadcn-svelte provides component library

### Mobile
Flutter with Dart for iOS and Android from single codebase
Riverpod for state management
Dio for HTTP client

### Backend
Go with Fiber or Chi framework
Single binary deployment
Handles similarity calculations and API endpoints

### Database
PostgreSQL as primary database
pgvector extension for cosine similarity calculations
sqlc for type-safe query generation

### Authentication
Lucia Auth integrates with SvelteKit for web
Supabase Auth as alternative managed option
Google and Apple OAuth providers

### Payments
Stripe SDK for subscription management
Webhook handlers for subscription lifecycle events

### Search
Fuse.js or MiniSearch for client-side fuzzy autocomplete
Search index preloaded on application start

### Internationalization
Paraglide (Inlang) for SvelteKit - compiles translations
flutter_localizations and intl package for mobile

## Infrastructure

### Hosting
Vercel or Cloudflare Pages for SvelteKit frontend
Fly.io or Railway for Go backend service
Railway or Supabase for managed PostgreSQL with pgvector

### Mobile CI/CD
Codemagic or Fastlane for Flutter builds
Automated submission to App Store and Google Play

## Third-Party Services
Stripe for payment processing
Google OAuth for authentication
Apple OAuth for authentication
USDA FoodData Central API for nutrition data
OpenFoodFacts API for nutrition data

## Data Flow
Web and mobile clients communicate with Go backend via REST API
Backend queries PostgreSQL for item data and similarity searches
pgvector performs cosine similarity directly in SQL queries
Search index is generated server-side and served to clients for local fuzzy search
Stripe webhooks notify backend of subscription changes

## Component Overview
Web app (SvelteKit) handles browser users
Mobile app (Flutter) handles iOS and Android users
API service (Go) provides unified backend for both clients
PostgreSQL stores users, items, saved lists, subscriptions
External APIs provide base nutrition data for admin curation
