interface RegistrationOptions {
	enabled: boolean;
}

/**
 * Registers the Phase 00 service worker when enabled and supported by the browser.
 *
 * @remarks Implements DESIGN-011 ServiceWorkerCache registration seam.
 */
export async function registerServiceWorker(options: RegistrationOptions): Promise<ServiceWorkerRegistration | null> {
	if (!options.enabled || typeof navigator === "undefined" || !("serviceWorker" in navigator)) {
		return null;
	}

	return navigator.serviceWorker.register("/service-worker.js");
}
