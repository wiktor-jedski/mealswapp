import { describe, expect, test } from "bun:test";
import { validateTask285Target } from "./task285-target";

// Implements DESIGN-014 LogAggregator deployed acceptance target regressions.
const acknowledged = "deployed-test-with-centralized-log-sink";

describe("Task 285 deployed target", () => {
	test("accepts an exact approved hostname only when every DNS result is public", async () => {
		await expect(
			validateTask285Target("https://acceptance.example.test", acknowledged, "acceptance.example.test", async () => [
				{ address: "8.8.8.8", family: 4 },
				{ address: "2606:4700:4700::1111", family: 6 }
			])
		).resolves.toBe("https://acceptance.example.test");
	});

	test("rejects any private, loopback, link-local, or reserved DNS result", async () => {
		for (const address of [
			"10.0.0.2",
			"100.64.0.2",
			"127.0.0.2",
			"169.254.2.3",
			"192.0.2.1",
			"224.0.0.1",
			"::1",
			"fe80::1",
			"2001:db8::1",
			"ff02::1"
		]) {
			await expect(
				validateTask285Target("https://acceptance.example.test", acknowledged, "acceptance.example.test", async () => [
					{ address: "8.8.8.8", family: 4 },
					{ address, family: address.includes(":") ? 6 : 4 }
				])
			).rejects.toThrow("publicly resolvable");
		}
	});

	test("rejects failed, empty, malformed, and IPv4-mapped DNS results", async () => {
		for (const resolver of [
			async () => [],
			async () => [{ address: "not-an-address", family: 4 }],
			async () => [{ address: "::ffff:127.0.0.1", family: 6 }],
			async () => {
				throw new Error("dns unavailable");
			}
		]) {
			await expect(
				validateTask285Target("https://acceptance.example.test", acknowledged, "acceptance.example.test", resolver)
			).rejects.toThrow("publicly resolvable");
		}
	});
});
