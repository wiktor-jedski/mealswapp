/// <reference types="vite/client" />

// Implements DESIGN-001 SearchView Vite environment typing for Svelte source diagnostics.
// Implements DESIGN-018 OAuthEntryPoint build-time provider allow-list.
interface ImportMetaEnv {
	readonly VITE_MEALSWAPP_OAUTH_PROVIDERS?: string;
}
