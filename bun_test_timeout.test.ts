import { describe, it, expect, mock } from "bun:test";

describe("timeout", () => {
	it("mocks setTimeout", () => {
		const originalSetTimeout = globalThis.setTimeout;
		let called = false;
		globalThis.setTimeout = ((fn: any) => {
			called = true;
			fn();
			return 1;
		}) as any;

		setTimeout(() => {}, 1000);
		expect(called).toBe(true);
		
		globalThis.setTimeout = originalSetTimeout;
	});
});
