#!/bin/sh
set -e
echo "Running database migrations..."
npx drizzle-kit migrate
echo "Starting application..."
node server.js
