#!/bin/bash
# Local verification script for Stripe CLI sandbox webhooks
# Implements DESIGN-007: StripeWebhookHandler verification commands

set -e

echo "=== Phase 06: Stripe CLI Sandbox Verification ==="

if ! command -v stripe &> /dev/null; then
    echo "Error: 'stripe' CLI is not installed. Please install it from https://stripe.com/docs/stripe-cli"
    exit 1
fi

echo "Ensure your backend is running locally at http://localhost:8080"
echo "You can start it with: bash scripts/start-services.sh && cd backend && go run ./cmd/api"
echo ""

echo "In a separate terminal, please run the following command to forward Stripe events:"
echo "  stripe listen --forward-to localhost:8080/api/v1/billing/webhook"
echo ""
echo "Press [Enter] when the listen command is running..."
read -r

echo ""
echo "[Test 1] Valid Signatures & Checkout Success"
echo "Running: stripe trigger checkout.session.completed"
stripe trigger checkout.session.completed
echo "Expected backend output: 200 OK, event processed, entitlement provisioned."
echo ""
sleep 2

echo "[Test 2] Duplicate Delivery Idempotency"
echo "Running: stripe trigger checkout.session.completed"
stripe trigger checkout.session.completed
echo "Expected backend output: 200 OK (duplicate recognized, no extra DB writes)."
echo ""
sleep 2

echo "[Test 3] Failed Payment State (Past Due)"
echo "Running: stripe trigger invoice.payment_failed"
stripe trigger invoice.payment_failed
echo "Expected backend output: 200 OK, entitlement marked as past_due."
echo ""
sleep 2

echo "[Test 4] Cancelled Subscription"
echo "Running: stripe trigger customer.subscription.deleted"
stripe trigger customer.subscription.deleted
echo "Expected backend output: 200 OK, entitlement marked as cancelled."
echo ""
sleep 2

echo "[Test 5] Invalid Signature Rejection"
echo "Running: curl -X POST http://localhost:8080/api/v1/billing/webhook -H \"Stripe-Signature: t=1614035650,v1=fake_signature\" -d \"{}\""
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/api/v1/billing/webhook \
  -H "Stripe-Signature: t=1614035650,v1=fake_signature" \
  -d "{}" | grep -q "400" && echo "Success: Received 400 Bad Request" || echo "Failed: Did not receive 400 Bad Request"
echo "Expected backend output: 400 Bad Request and logged security event."
echo ""

echo "Verification complete! Review the backend logs to confirm all expected behaviors."
echo "No real Stripe keys or customer data should have been committed or processed."
