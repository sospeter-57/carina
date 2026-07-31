#!/usr/bin/env bash

# migrate.sh - Apply Postgres migrations
#
# Usage:
#   ./scripts/migrate.sh [up|down|status|create NAME]
#
# Environment:
#   DATABASE_URL must be set (or passed via .env)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default command
COMMAND="${1:-up}"
MIGRATION_NAME="${2:-}"

# Determine script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/internal/platform/postgres/migrations"

# Load .env if present
if [ -f "$PROJECT_ROOT/.env" ]; then
    set -a
    source "$PROJECT_ROOT/.env"
    set +a
fi

# Validate DATABASE_URL
if [ -z "${DATABASE_URL:-}" ]; then
    echo -e "${RED}Error: DATABASE_URL is not set${NC}"
    echo "Please set DATABASE_URL in your environment or .env file"
    exit 1
fi

# Mask password for logging
MASKED_URL="$(echo "$DATABASE_URL" | sed 's/:[^:@]*@/:****@/')"

# Use golang-migrate via Docker unless otherwise specified
MIGRATE_CMD="${MIGRATE_CMD:-docker run --rm -v "$MIGRATIONS_DIR:/migrations" --network host migrate/migrate:v4.17.0 -path=/migrations -database="$DATABASE_URL"}"

log_info() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if we have migration files
if [ ! -d "$MIGRATIONS_DIR" ]; then
    log_error "Migrations directory not found: $MIGRATIONS_DIR"
    exit 1
fi

case "$COMMAND" in
    up)
        echo "Applying migrations to $MASKED_URL"
        $MIGRATE_CMD up
        log_info "Migrations applied successfully"
        ;;

    down)
        if [ -z "$MIGRATION_NAME" ]; then
            echo "Rolling back 1 migration"
            $MIGRATE_CMD down 1
        else
            echo "Rolling back to $MIGRATION_NAME"
            $MIGRATE_CMD goto "$MIGRATION_NAME"
        fi
        log_info "Rollback complete"
        ;;

    status)
        echo "Migration status for $MASKED_URL"
        $MIGRATE_CMD version
        ;;

    create)
        if [ -z "$MIGRATION_NAME" ]; then
            log_error "create command requires a migration name"
            echo "Usage: $0 create <name>"
            exit 1
        fi
        TIMESTAMP=$(date +%Y%m%d%H%M%S)
        UP_FILE="$MIGRATIONS_DIR/${TIMESTAMP}_${MIGRATION_NAME}.up.sql"
        DOWN_FILE="$MIGRATIONS_DIR/${TIMESTAMP}_${MIGRATION_NAME}.down.sql"

        cat > "$UP_FILE" <<EOF
-- ${TIMESTAMP}_${MIGRATION_NAME}.up.sql
-- Add your migration SQL here
EOF

        cat > "$DOWN_FILE" <<EOF
-- ${TIMESTAMP}_${MIGRATION_NAME}.down.sql
-- Add your rollback SQL here
EOF

        log_info "Created migration files:"
        echo "  - $UP_FILE"
        echo "  - $DOWN_FILE"
        ;;

    force)
        if [ -z "$MIGRATION_NAME" ]; then
            log_error "force command requires a version number"
            echo "Usage: $0 force <version>"
            exit 1
        fi
        $MIGRATE_CMD force "$MIGRATION_NAME"
        log_info "Forced version to $MIGRATION_NAME"
        ;;

    *)
        echo "Usage: $0 [up|down|status|create NAME|force VERSION]"
        echo ""
        echo "Commands:"
        echo "  up            Apply all pending migrations"
        echo "  down          Roll back the last migration"
        echo "  status        Show current migration version"
        echo "  create NAME   Create new migration files"
        echo "  force VERSION Force a specific version (use with caution)"
        exit 1
        ;;
esac