-- Implements DESIGN-006 AccountLockoutTracker failure counter query.
WITH current_state AS (
    SELECT failed_login_count,
           locked_until,
           CASE
               WHEN locked_until IS NOT NULL AND locked_until <= $4 THEN 1
               ELSE failed_login_count + 1
           END AS next_count
    FROM users
    WHERE id = $1
), updated AS (
    UPDATE users
    SET failed_login_count = current_state.next_count,
        locked_until = CASE
            WHEN current_state.next_count >= $2 THEN $3
            WHEN current_state.locked_until IS NOT NULL AND current_state.locked_until <= $4 THEN NULL
            ELSE current_state.locked_until
        END,
        updated_at = now()
    FROM current_state
    WHERE users.id = $1
    RETURNING users.failed_login_count, users.locked_until
)
SELECT failed_login_count, locked_until
FROM updated;
