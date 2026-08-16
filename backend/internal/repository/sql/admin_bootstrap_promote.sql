-- Implements DESIGN-009 AdminController atomic administrator promotion.
UPDATE users
SET role = 'admin',
    updated_at = now()
WHERE id = $1
  AND role = 'user';
