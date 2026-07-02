#!/usr/bin/env python3

# Implements DESIGN-007 StripeWebhookHandler local Stripe CLI sandbox verification.

import argparse
import hashlib
import hmac
import json
import os
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_WEBHOOK_URL = "http://127.0.0.1:8080/api/v1/billing/stripe/webhook"
DEFAULT_SECRET = "whsec_local_fixture"
DEFAULT_USER_ID = "11111111-1111-4111-8111-111111111171"
FIXTURE_CUSTOMER_ID = "cus_test_task171"
DEAD_LETTER_CUSTOMER_ID = "cus_test_task171_deadletter"
FIXTURE_SUBSCRIPTION_ID = "sub_test_task171"


@dataclass(frozen=True)
class FixtureEvent:
	event_id: str
	event_type: str
	status: str = ""
	customer_id: str = FIXTURE_CUSTOMER_ID
	subscription_id: str = FIXTURE_SUBSCRIPTION_ID


FIXTURE_EVENTS = (
	FixtureEvent("evt_task171_checkout_completed", "checkout.session.completed"),
	FixtureEvent("evt_task171_payment_failed", "invoice.payment_failed"),
	FixtureEvent("evt_task171_subscription_deleted", "customer.subscription.deleted"),
	FixtureEvent("evt_task171_retry_dead_letter", "invoice.payment_failed", customer_id=DEAD_LETTER_CUSTOMER_ID),
)


def fixture_payload(event: FixtureEvent, user_id: str) -> bytes:
	object_id = "cs_test_task171"
	subscription_id = event.subscription_id
	if event.event_type.startswith("customer.subscription."):
		object_id = event.subscription_id
		subscription_id = ""
	payload = {
		"id": event.event_id,
		"type": event.event_type,
		"data": {
			"object": {
				"id": object_id,
				"client_reference_id": user_id,
				"customer": event.customer_id,
				"subscription": subscription_id,
				"status": event.status,
				"metadata": {"user_id": user_id},
			}
		},
	}
	return json.dumps(payload, separators=(",", ":"), sort_keys=True).encode("utf-8")


def stripe_signature(payload: bytes, secret: str, timestamp: int | None = None) -> str:
	timestamp = timestamp or int(time.time())
	signed_payload = f"{timestamp}.".encode("utf-8") + payload
	digest = hmac.new(secret.encode("utf-8"), signed_payload, hashlib.sha256).hexdigest()
	return f"t={timestamp},v1={digest}"


def post_webhook(url: str, payload: bytes, signature: str) -> tuple[int, str]:
	request = urllib.request.Request(
		url,
		data=payload,
		method="POST",
		headers={"Content-Type": "application/json", "Stripe-Signature": signature},
	)
	try:
		with urllib.request.urlopen(request, timeout=10) as response:
			return response.status, response.read().decode("utf-8")
	except urllib.error.HTTPError as exc:
		return exc.code, exc.read().decode("utf-8")


def require_status(name: str, got: int, want: int, body: str) -> None:
	if got != want:
		raise RuntimeError(f"{name}: HTTP {got}, want {want}; response: {body}")


def run_psql(database_url: str, sql: str) -> str:
	result = subprocess.run(
		["psql", database_url, "-v", "ON_ERROR_STOP=1", "-Atc", sql],
		cwd=ROOT,
		text=True,
		capture_output=True,
	)
	if result.returncode != 0:
		raise RuntimeError(f"psql failed: {result.stderr.strip() or result.stdout.strip()}")
	return result.stdout.strip()


def prepare_database(database_url: str, user_id: str) -> None:
	event_ids = ", ".join(f"'{event.event_id}'" for event in FIXTURE_EVENTS)
	sql = (
		"ALTER TABLE entitlements DROP CONSTRAINT IF EXISTS task171_dead_letter_failure; "
		"INSERT INTO users (id) VALUES "
		f"('{user_id}') ON CONFLICT (id) DO NOTHING; "
		f"DELETE FROM stripe_dead_letters WHERE event_id IN ({event_ids}); "
		f"DELETE FROM processed_stripe_events WHERE event_id IN ({event_ids}); "
		f"DELETE FROM entitlements WHERE user_id = '{user_id}' "
		"AND stripe_customer_id LIKE 'cus_test_task171%';"
	)
	run_psql(database_url, sql)


def latest_entitlement_status(database_url: str, user_id: str) -> str:
	sql = (
		"SELECT tier || ':' || status || ':' || coalesce(stripe_customer_id, '') || ':' || "
		"coalesce(stripe_subscription_id, '') FROM entitlements "
		f"WHERE user_id = '{user_id}' ORDER BY created_at DESC LIMIT 1;"
	)
	return run_psql(database_url, sql)


def entitlement_count(database_url: str, user_id: str) -> str:
	sql = f"SELECT count(*) FROM entitlements WHERE user_id = '{user_id}' AND stripe_customer_id LIKE 'cus_test_task171%';"
	return run_psql(database_url, sql)


def dead_letter_status(database_url: str) -> str:
	sql = (
		"SELECT event_id || ':' || event_type || ':' || failure_category || ':' || "
		"stripe_customer_id || ':' || stripe_subscription_id || ':' || "
		"CASE WHEN payload_sha256 ~ '^[0-9a-f]{64}$' THEN 'hash_ok' ELSE 'hash_bad' END "
		"FROM stripe_dead_letters WHERE event_id = 'evt_task171_retry_dead_letter' "
		"ORDER BY created_at DESC LIMIT 1;"
	)
	return run_psql(database_url, sql)


def add_dead_letter_failure_constraint(database_url: str) -> None:
	sql = (
		"ALTER TABLE entitlements DROP CONSTRAINT IF EXISTS task171_dead_letter_failure; "
		"ALTER TABLE entitlements ADD CONSTRAINT task171_dead_letter_failure "
		f"CHECK (stripe_customer_id <> '{DEAD_LETTER_CUSTOMER_ID}');"
	)
	run_psql(database_url, sql)


def drop_dead_letter_failure_constraint(database_url: str) -> None:
	run_psql(database_url, "ALTER TABLE entitlements DROP CONSTRAINT IF EXISTS task171_dead_letter_failure;")


def print_stripe_cli_commands(url: str) -> None:
	print("Stripe CLI command sequence:")
	print(f"  stripe listen --forward-to {url}")
	print("  export MEALSWAPP_STRIPE_WEBHOOK_SECRET='<whsec value printed by stripe listen>'")
	print("  stripe trigger checkout.session.completed")
	print("  stripe trigger invoice.payment_failed")
	print("  stripe trigger customer.subscription.deleted")
	print()
	print("Use the generated webhook secret only for the local process. Do not commit it.")


def print_fixture_curl(url: str, secret: str, user_id: str) -> None:
	event = FIXTURE_EVENTS[0]
	payload = fixture_payload(event, user_id)
	signature = stripe_signature(payload, secret)
	print("Deterministic signed fixture example:")
	print(f"  curl -i -X POST {url} \\")
	print("    -H 'Content-Type: application/json' \\")
	print(f"    -H 'Stripe-Signature: {signature}' \\")
	print(f"    --data '{payload.decode('utf-8')}'")


def verify_no_sensitive_values(secret: str, payloads: list[bytes]) -> None:
	joined = "\n".join([secret, *(payload.decode("utf-8") for payload in payloads)])
	for blocked in ("sk_live_", "rk_live_", "4242424242424242", "@"):
		if blocked in joined:
			raise RuntimeError(f"sensitive or real-payment-looking value found in fixtures: {blocked}")


def run_http_verification(url: str, secret: str, user_id: str, database_url: str | None) -> None:
	live_events = FIXTURE_EVENTS[:3]
	dead_letter_event = FIXTURE_EVENTS[3]
	payloads = [fixture_payload(event, user_id) for event in FIXTURE_EVENTS]
	verify_no_sensitive_values(secret, payloads)
	if database_url:
		if not shutil.which("psql"):
			raise RuntimeError("--database-url requires psql on PATH")
		prepare_database(database_url, user_id)
		print("PASS local database prepared with deterministic task-171 fixture user")

	checkout_payload = fixture_payload(live_events[0], user_id)
	status, body = post_webhook(url, checkout_payload, stripe_signature(checkout_payload, secret))
	require_status("valid checkout signature", status, 200, body)
	print("PASS valid signed checkout event accepted")
	if database_url:
		latest = latest_entitlement_status(database_url, user_id)
		want = f"paid:active:{FIXTURE_CUSTOMER_ID}:{FIXTURE_SUBSCRIPTION_ID}"
		if latest != want:
			raise RuntimeError(f"latest entitlement after checkout = {latest!r}, want {want!r}")
		first_count = entitlement_count(database_url, user_id)
		if first_count != "1":
			raise RuntimeError(f"entitlement count after checkout = {first_count!r}, want '1'")
		print("PASS local database latest entitlement is paid:active after checkout")

	status, body = post_webhook(url, checkout_payload, "t=1,v1=invalid")
	require_status("invalid signature", status, 400, body)
	print("PASS invalid signature rejected")

	status, body = post_webhook(url, checkout_payload, stripe_signature(checkout_payload, secret))
	require_status("duplicate checkout delivery", status, 200, body)
	if '"duplicate":true' not in body.replace(" ", ""):
		raise RuntimeError(f"duplicate checkout delivery did not report duplicate=true: {body}")
	print("PASS duplicate provider event accepted without duplicate side effects")
	if database_url:
		duplicate_count = entitlement_count(database_url, user_id)
		if duplicate_count != "1":
			raise RuntimeError(f"entitlement count after duplicate = {duplicate_count!r}, want '1'")
		print("PASS duplicate delivery left entitlement history count unchanged")

	failed_payload = fixture_payload(live_events[1], user_id)
	status, body = post_webhook(url, failed_payload, stripe_signature(failed_payload, secret))
	require_status("payment failed event", status, 200, body)
	print("PASS payment failure event accepted for past_due projection")
	if database_url:
		latest = latest_entitlement_status(database_url, user_id)
		want = f"paid:past_due:{FIXTURE_CUSTOMER_ID}:{FIXTURE_SUBSCRIPTION_ID}"
		if latest != want:
			raise RuntimeError(f"latest entitlement after payment failure = {latest!r}, want {want!r}")
		print("PASS local database latest entitlement is paid:past_due after failed payment")

	cancelled_payload = fixture_payload(live_events[2], user_id)
	status, body = post_webhook(url, cancelled_payload, stripe_signature(cancelled_payload, secret))
	require_status("subscription deleted event", status, 200, body)
	print("PASS subscription deletion event accepted for cancelled projection")

	if database_url:
		latest = latest_entitlement_status(database_url, user_id)
		want = f"paid:cancelled:{FIXTURE_CUSTOMER_ID}:{FIXTURE_SUBSCRIPTION_ID}"
		if latest != want:
			raise RuntimeError(f"latest entitlement = {latest!r}, want {want!r}")
		print("PASS local database latest entitlement is paid:cancelled after failed/cancelled fixtures")

	if database_url:
		add_dead_letter_failure_constraint(database_url)
		try:
			dead_letter_payload = fixture_payload(dead_letter_event, user_id)
			status, body = post_webhook(url, dead_letter_payload, stripe_signature(dead_letter_payload, secret))
			require_status("retry/dead-letter event", status, 500, body)
			print("PASS forced entitlement write failure returned 500 for Stripe retry")
			dead_letter = dead_letter_status(database_url)
			want = (
				"evt_task171_retry_dead_letter:invoice.payment_failed:webhook_processing_failed:"
				f"{DEAD_LETTER_CUSTOMER_ID}:{FIXTURE_SUBSCRIPTION_ID}:hash_ok"
			)
			if dead_letter != want:
				raise RuntimeError(f"dead letter = {dead_letter!r}, want {want!r}")
			print("PASS sanitized dead-letter metadata persisted for retry-producing failure")
		finally:
			drop_dead_letter_failure_constraint(database_url)


def main() -> int:
	parser = argparse.ArgumentParser(description="Verify local Stripe webhook sandbox behavior for Phase 06 task 171.")
	parser.add_argument("--webhook-url", default=os.getenv("MEALSWAPP_STRIPE_WEBHOOK_URL", DEFAULT_WEBHOOK_URL))
	parser.add_argument("--webhook-secret", default=os.getenv("MEALSWAPP_STRIPE_WEBHOOK_SECRET", DEFAULT_SECRET))
	parser.add_argument("--user-id", default=os.getenv("MEALSWAPP_STRIPE_VERIFY_USER_ID", DEFAULT_USER_ID))
	parser.add_argument("--database-url", default=os.getenv("MEALSWAPP_DATABASE_URL", ""))
	parser.add_argument("--commands-only", action="store_true", help="print Stripe CLI/curl commands without sending events")
	args = parser.parse_args()

	if not args.webhook_secret.startswith("whsec_"):
		raise SystemExit("MEALSWAPP_STRIPE_WEBHOOK_SECRET must be a Stripe webhook signing secret placeholder or sandbox secret")

	print_stripe_cli_commands(args.webhook_url)
	print()
	print_fixture_curl(args.webhook_url, args.webhook_secret, args.user_id)
	if args.commands_only:
		return 0

	run_http_verification(args.webhook_url, args.webhook_secret, args.user_id, args.database_url or None)
	print("Stripe sandbox webhook verification passed.")
	return 0


if __name__ == "__main__":
	sys.exit(main())
