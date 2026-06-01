-- Implements DESIGN-007 EntitlementManager integration fixture.
SELECT count(*) FROM entitlements WHERE user_id = $1;
