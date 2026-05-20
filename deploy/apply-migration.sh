#!/bin/bash

for migration in db/migrations/*.up.sql; do
  echo "Applying ${migration}"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
done
