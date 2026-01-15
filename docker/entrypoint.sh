#!/bin/bash
set -e

# =============================================================================
# Entrypoint script for Whatomate unified container
# Parses DATABASE_URL if provided and sets individual environment variables
# =============================================================================

echo "========================================"
echo "  Whatomate - Starting up..."
echo "========================================"

# Parse DATABASE_URL if provided
# Format: postgresql://user:password@host:port/dbname?sslmode=value&other_params
if [ -n "$DATABASE_URL" ]; then
    echo "Parsing DATABASE_URL..."
    
    # Remove the protocol prefix
    url_without_protocol="${DATABASE_URL#postgresql://}"
    url_without_protocol="${url_without_protocol#postgres://}"
    
    # Extract user:password part (before @)
    user_pass="${url_without_protocol%%@*}"
    
    # Extract host:port/dbname?params (after @)
    host_db="${url_without_protocol#*@}"
    
    # Extract user and password
    DB_USER="${user_pass%%:*}"
    DB_PASSWORD="${user_pass#*:}"
    
    # Extract host:port and dbname?params
    host_port="${host_db%%/*}"
    db_params="${host_db#*/}"
    
    # Extract host and port
    if [[ "$host_port" == *":"* ]]; then
        DB_HOST="${host_port%%:*}"
        DB_PORT="${host_port#*:}"
    else
        DB_HOST="$host_port"
        DB_PORT="5432"
    fi
    
    # Extract database name and params
    DB_NAME="${db_params%%\?*}"
    params="${db_params#*\?}"
    
    # Parse sslmode from params
    DB_SSLMODE="disable"
    if [[ "$params" == *"sslmode="* ]]; then
        # Extract sslmode value
        sslmode_part="${params#*sslmode=}"
        DB_SSLMODE="${sslmode_part%%&*}"
    fi
    
    # Set environment variables for Whatomate
    export WHATOMATE_DATABASE_HOST="$DB_HOST"
    export WHATOMATE_DATABASE_PORT="$DB_PORT"
    export WHATOMATE_DATABASE_USER="$DB_USER"
    export WHATOMATE_DATABASE_PASSWORD="$DB_PASSWORD"
    export WHATOMATE_DATABASE_NAME="$DB_NAME"
    export WHATOMATE_DATABASE_SSL_MODE="$DB_SSLMODE"
    
    echo "Database configured:"
    echo "  Host: $DB_HOST"
    echo "  Port: $DB_PORT"
    echo "  User: $DB_USER"
    echo "  Database: $DB_NAME"
    echo "  SSL Mode: $DB_SSLMODE"
fi

# Wait for Redis to be ready (started by supervisor)
wait_for_redis() {
    echo "Waiting for Redis to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if redis-cli -h 127.0.0.1 -p 6379 ping > /dev/null 2>&1; then
            echo "Redis is ready!"
            return 0
        fi
        echo "  Attempt $attempt/$max_attempts - Redis not ready yet..."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo "Warning: Redis may not be ready, proceeding anyway..."
    return 0
}

# Generate config.toml from environment variables if it doesn't exist or is default
generate_config() {
    echo "Generating configuration..."
    
    cat > /app/config.toml << EOF
# Whatomate Configuration - Auto-generated
# Environment variables will override these values

[app]
name = "${WHATOMATE_APP_NAME:-Whatomate}"
environment = "${WHATOMATE_APP_ENVIRONMENT:-production}"
debug = ${WHATOMATE_APP_DEBUG:-false}

[server]
host = "${WHATOMATE_SERVER_HOST:-0.0.0.0}"
port = ${WHATOMATE_SERVER_PORT:-8080}
read_timeout = ${WHATOMATE_SERVER_READ_TIMEOUT:-30}
write_timeout = ${WHATOMATE_SERVER_WRITE_TIMEOUT:-30}
base_path = "${WHATOMATE_SERVER_BASE_PATH:-}"

[database]
host = "${WHATOMATE_DATABASE_HOST:-localhost}"
port = ${WHATOMATE_DATABASE_PORT:-5432}
user = "${WHATOMATE_DATABASE_USER:-whatomate}"
password = "${WHATOMATE_DATABASE_PASSWORD:-whatomate}"
name = "${WHATOMATE_DATABASE_NAME:-whatomate}"
ssl_mode = "${WHATOMATE_DATABASE_SSL_MODE:-disable}"
max_open_conns = ${WHATOMATE_DATABASE_MAX_OPEN_CONNS:-25}
max_idle_conns = ${WHATOMATE_DATABASE_MAX_IDLE_CONNS:-5}
conn_max_lifetime = ${WHATOMATE_DATABASE_CONN_MAX_LIFETIME:-300}

[redis]
host = "${WHATOMATE_REDIS_HOST:-127.0.0.1}"
port = ${WHATOMATE_REDIS_PORT:-6379}
password = "${WHATOMATE_REDIS_PASSWORD:-}"
db = ${WHATOMATE_REDIS_DB:-0}

[jwt]
secret = "${WHATOMATE_JWT_SECRET:-change-this-in-production-$(date +%s)}"
access_expiry_mins = ${WHATOMATE_JWT_ACCESS_EXPIRY_MINS:-15}
refresh_expiry_days = ${WHATOMATE_JWT_REFRESH_EXPIRY_DAYS:-7}

[storage]
type = "${WHATOMATE_STORAGE_TYPE:-local}"
local_path = "${WHATOMATE_STORAGE_LOCAL_PATH:-/app/uploads}"
s3_bucket = "${WHATOMATE_STORAGE_S3_BUCKET:-}"
s3_region = "${WHATOMATE_STORAGE_S3_REGION:-}"
s3_key = "${WHATOMATE_STORAGE_S3_KEY:-}"
s3_secret = "${WHATOMATE_STORAGE_S3_SECRET:-}"

[whatsapp]
webhook_verify_token = "${WHATOMATE_WHATSAPP_WEBHOOK_VERIFY_TOKEN:-}"
api_version = "${WHATOMATE_WHATSAPP_API_VERSION:-v18.0}"
base_url = "${WHATOMATE_WHATSAPP_BASE_URL:-https://graph.facebook.com}"

[ai]
openai_key = "${WHATOMATE_AI_OPENAI_KEY:-}"
anthropic_key = "${WHATOMATE_AI_ANTHROPIC_KEY:-}"
google_key = "${WHATOMATE_AI_GOOGLE_KEY:-}"
EOF

    echo "Configuration generated successfully!"
}

# Generate config
generate_config

echo ""
echo "Starting services via supervisor..."
echo "========================================"

# Start supervisor (which starts Redis and Whatomate)
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
