import { afterEach, expect, test } from "bun:test";
import {
	createAdminClassification, createAdminItem, deleteAdminClassification, deleteAdminItem, getAdminItem,
	listAdminClassifications, lookupAdminUsers, replaceAdminClassification, replaceAdminItem, retryAdminDeletion
} from "./admin-client";
import type { AdminItemRequest } from "./generated";

// Implements DESIGN-009 generated-contract client CRUD, lookup, retry, and safe failure verification.

const originalFetch = globalThis.fetch;
const itemId = "00000000-0000-4000-8000-000000000001";
const classId = "00000000-0000-4000-8000-000000000002";
const parentClassId = "00000000-0000-4000-8000-000000000005";
const userId = "00000000-0000-4000-8000-000000000003";
const requestId = "00000000-0000-4000-8000-000000000004";
const request: AdminItemRequest = { name: "Rice", physicalState: "solid", macrosPer100: { protein: 2, carbohydrates: 28, fat: 0 }, micros: {}, foodCategoryIds: [], culinaryRoleIds: [] };
const item = { ...request, id: itemId, prepTimeMinutes: 0, foodCategories: [], culinaryRoles: [] };

afterEach(() => { globalThis.fetch = originalFetch; });
const envelope = (data: unknown) => ({ status: "ok", requestId: "task-256", data });
const response = (status: number, data?: unknown) => data === undefined ? new Response(null, { status }) : new Response(JSON.stringify(data), { status, headers: { "Content-Type": "application/json" } });

test("uses documented generated-contract routes, methods, CSRF, and idempotency", async () => {
	const calls: Array<{ url: string; init: RequestInit }> = [];
	const queued = [response(200, envelope(item)), response(201, envelope(item)), response(200, envelope(item)), response(204), response(200, envelope({ classifications: [{ id: classId, name: "Staple", kind: "food_category" }] })), response(201, envelope({ classification: { id: classId, name: "Staple", kind: "food_category" } })), response(200, envelope({ classification: { id: classId, name: "Base", kind: "food_category" } })), response(204), response(200, envelope({ users: [{ id: userId, email: "user@example.test", emailVerified: true, createdAt: "2026-07-21T00:00:00Z" }] })), response(200, envelope({ requestId, status: "pending" }))];
	globalThis.fetch = ((input: string | URL | Request, init = {}) => { calls.push({ url: String(input), init }); return Promise.resolve(queued.shift()!); }) as typeof fetch;

	await getAdminItem(itemId); await createAdminItem(request, "stable-key", { csrfToken: "csrf" }); await replaceAdminItem(itemId, request, { csrfToken: "csrf" }); await deleteAdminItem(itemId, { csrfToken: "csrf" });
	await listAdminClassifications("food_category"); await createAdminClassification("food_category", { name: "Staple" }, { csrfToken: "csrf" }); await replaceAdminClassification(classId, { name: "Base" }, { csrfToken: "csrf" }); await deleteAdminClassification(classId, { csrfToken: "csrf" });
	await lookupAdminUsers({ email: "user+tag@example.test", limit: 1 }); await retryAdminDeletion(userId, requestId, { csrfToken: "csrf" });

	expect(calls[1]).toMatchObject({ url: "/api/v1/admin/items", init: { method: "POST" } });
	expect(calls[1]!.init.headers).toMatchObject({ "Idempotency-Key": "stable-key", "X-CSRF-Token": "csrf" });
	expect(calls[4]!.url).toBe("/api/v1/admin/classifications?kind=food_category");
	expect(calls[8]!.url).toContain("email=user%2Btag%40example.test");
	expect(calls[9]!.url).toBe(`/api/v1/admin/users/${userId}/deletion-requests/${requestId}/retry`);
});

test("rejects conflicts, audit failures, malformed privacy projections, and false-success statuses", async () => {
	const queued = [
		response(409, { status: "error", requestId: "r", error: { code: "classification_in_use" } }),
		response(500, { status: "error", requestId: "r", error: { code: "audit_write_failed", message: "internal snapshot" } }),
		response(200, envelope({ users: [{ id: userId, email: "user@example.test", emailVerified: true, createdAt: "2026-07-21T00:00:00Z", password: "secret" }] })),
		response(200, envelope(item))
	];
	globalThis.fetch = (() => Promise.resolve(queued.shift()!)) as typeof fetch;
	await expect(deleteAdminClassification(classId, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 409, appError: { code: "classification_in_use" } });
	await expect(replaceAdminItem(itemId, request, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 500, appError: { code: "audit_write_failed", message: expect.not.stringContaining("snapshot") } });
	// Unknown user fields are not part of the generated privacy-minimized contract.
	await expect(lookupAdminUsers({ userId })).rejects.toMatchObject({ appError: { code: "malformed_admin_response" } });
	await expect(createAdminItem(request, "stable-key", { csrfToken: "csrf" })).rejects.toMatchObject({ appError: { code: "malformed_admin_response" } });
});

test("strictly bounds and validates nested item, classification, and user projections", async () => {
	const liquid = { ...item, physicalState: "liquid", densityGramsPerMilliliter: 1.2, densitySourceKind: "manual", macrosPer100: { protein: 10, carbohydrates: 110, fat: 5 }, foodCategories: [{ id: classId, name: "Staple", kind: "food_category" }] };
	const queued = [
		response(200, envelope(liquid)),
		response(200, envelope({ ...item, macrosPer100: { protein: "x", carbohydrates: 1, fat: 1 } })),
		response(200, envelope({ ...item, micros: { sodium: "x" } })),
		response(200, envelope({ ...item, imageUrl: "javascript:alert(1)" })),
		response(200, envelope({ classifications: Array.from({ length: 1001 }, () => ({ id: classId, name: "Staple", kind: "food_category" })) })),
		response(200, envelope({ users: [{ id: userId, email: "user@example.test", emailVerified: true, createdAt: "2026-07-21T00:00:00Z", deletion: { requestId, status: "failed", failureCategory: "internal", retryCount: -1, requestedAt: "not-a-date" } }] }))
	];
	globalThis.fetch = (() => Promise.resolve(queued.shift()!)) as typeof fetch;
	expect((await getAdminItem(itemId)).macrosPer100.carbohydrates).toBe(110);
	for (const operation of [() => getAdminItem(itemId), () => getAdminItem(itemId), () => getAdminItem(itemId), () => listAdminClassifications("food_category"), () => lookupAdminUsers({ userId })]) {
		await expect(operation()).rejects.toMatchObject({ appError: { code: "malformed_admin_response" } });
	}
});

test("accepts only calendar-valid RFC3339 admin user dates", async () => {
	const user = (createdAt: string, requestedAt = "2024-02-29T23:59:59.123456789+02:30") => ({
		id: userId,
		email: "user@example.test",
		emailVerified: true,
		createdAt,
		deletion: { requestId, status: "failed", failureCategory: "transient", retryCount: 1, requestedAt }
	});
	const valid = ["2024-02-29T23:59:59Z", "2026-07-21T00:00:00-07:00", "2026-07-21T00:00:00.123456789+02:30"];
	const invalid = [
		"2026-02-30T00:00:00Z",
		"2025-02-29T00:00:00Z",
		"2026-04-31T00:00:00Z",
		"2026-07-21 00:00:00Z",
		"2026-07-21t00:00:00Z",
		"2026-07-21T00:00:00",
		"2026-07-21T00:00:00+0200",
		"2026-07-21T00:00:00+24:00",
		"2026-07-21T00:00:00+02:60"
	];
	const queued = [
		...valid.map((createdAt) => response(200, envelope({ users: [user(createdAt)] }))),
		...invalid.map((createdAt) => response(200, envelope({ users: [user(createdAt)] }))),
		response(200, envelope({ users: [user("2026-07-21T00:00:00Z", "2026-02-30T00:00:00Z")] }))
	];
	globalThis.fetch = (() => Promise.resolve(queued.shift()!)) as typeof fetch;

	for (const createdAt of valid) expect((await lookupAdminUsers({ userId })).users[0]!.createdAt).toBe(createdAt);
	for (const _ of invalid) await expect(lookupAdminUsers({ userId })).rejects.toMatchObject({ appError: { code: "malformed_admin_response" } });
	await expect(lookupAdminUsers({ userId })).rejects.toMatchObject({ appError: { code: "malformed_admin_response" } });
});

test("bounds request and response bodies, allowlists error codes, and preserves cancellation", async () => {
	await expect(createAdminItem({ ...request, imageUrl: "x".repeat(70_000) }, "stable-key", { csrfToken: "csrf" })).rejects.toMatchObject({ appError: { code: "invalid_admin_request" } });
	globalThis.fetch = (() => Promise.resolve(new Response(JSON.stringify(envelope(item)) + " ".repeat(300_000), { status: 200 }))) as typeof fetch;
	await expect(getAdminItem(itemId)).rejects.toMatchObject({ appError: { code: "malformed_admin_response" } });
	globalThis.fetch = (() => Promise.resolve(response(500, { error: { code: "internal_snapshot_secret" } }))) as typeof fetch;
	await expect(getAdminItem(itemId)).rejects.toMatchObject({ status: 500, appError: { code: "admin_request_failed" } });
	const abort = new DOMException("cancelled", "AbortError");
	globalThis.fetch = (() => Promise.reject(abort)) as typeof fetch;
	await expect(getAdminItem(itemId, new AbortController().signal)).rejects.toBe(abort);
});

test("cancels declared oversized success and error response bodies without changing safe errors", async () => {
	const canceled = { success: false, error: false };
	const oversized = (status: number, kind: keyof typeof canceled, cancelFails = false) => new Response(new ReadableStream({
		cancel() {
			canceled[kind] = true;
			if (cancelFails) throw new Error("transport cancel detail");
		}
	}), { status, headers: { "Content-Length": "300000" } });
	const queued = [oversized(200, "success", true), oversized(500, "error")];
	globalThis.fetch = (() => Promise.resolve(queued.shift()!)) as typeof fetch;

	await expect(getAdminItem(itemId)).rejects.toMatchObject({ status: 200, appError: { code: "malformed_admin_response", message: expect.not.stringContaining("transport") } });
	expect(canceled.success).toBeTrue();
	await expect(getAdminItem(itemId)).rejects.toMatchObject({ status: 500, appError: { code: "admin_request_failed", message: expect.not.stringContaining("transport") } });
	expect(canceled.error).toBeTrue();
});

test("cancels wrong-status successful JSON and empty mutation bodies without changing safe errors", async () => {
	const canceled = { json: false, jsonFailure: false, empty: false, emptyFailure: false };
	const wrongStatus = (status: number, kind: keyof typeof canceled, cancelFails = false) => new Response(new ReadableStream({
		cancel() {
			canceled[kind] = true;
			if (cancelFails) throw new Error("transport cancel detail");
		}
	}), { status });
	const queued = [wrongStatus(201, "json"), wrongStatus(201, "jsonFailure", true), wrongStatus(200, "empty"), wrongStatus(200, "emptyFailure", true)];
	globalThis.fetch = (() => Promise.resolve(queued.shift()!)) as typeof fetch;

	await expect(getAdminItem(itemId)).rejects.toMatchObject({ status: 201, appError: { code: "malformed_admin_response" } });
	await expect(replaceAdminItem(itemId, request, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 201, appError: { code: "malformed_admin_response", message: expect.not.stringContaining("transport") } });
	await expect(deleteAdminItem(itemId, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 200, appError: { code: "malformed_admin_response" } });
	await expect(deleteAdminClassification(classId, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 200, appError: { code: "malformed_admin_response", message: expect.not.stringContaining("transport") } });
	expect(canceled).toEqual({ json: true, jsonFailure: true, empty: true, emptyFailure: true });
});

test("preserves a classification parent in the generated-contract replacement request and response", async () => {
	let body: unknown;
	globalThis.fetch = ((_, init = {}) => {
		body = JSON.parse(String(init.body));
		return Promise.resolve(response(200, envelope({ classification: { id: classId, name: "Leaf", kind: "food_category", parentId: parentClassId } })));
	}) as typeof fetch;

	await expect(replaceAdminClassification(classId, { name: "Leaf", parentId: parentClassId }, { csrfToken: "csrf" })).resolves.toMatchObject({ id: classId, parentId: parentClassId });
	expect(body).toEqual({ name: "Leaf", parentId: parentClassId });
});
