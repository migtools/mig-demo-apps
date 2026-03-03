#!/bin/bash
set -e

CDIR="/opt/todolist"
DB_BACKEND="${DB_BACKEND:-mariadb}"

# -----------------------------------------------------------------------
# Determine whether to start a local DB or use an external one.
# External = MYSQL_HOST set to anything other than 127.0.0.1/localhost
#            MONGO_URI  set to anything that doesn't point at 127.0.0.1/localhost
# -----------------------------------------------------------------------
skip_local_db() {
    if [ "$DB_BACKEND" = "mongodb" ]; then
        if [ -n "$MONGO_URI" ]; then
            # Match both authenticated  mongodb://user:pass@127.0.0.1/...
            # and unauthenticated       mongodb://127.0.0.1/...
            echo "$MONGO_URI" | grep -qE '(://|@)(127\.0\.0\.1|localhost)(:|/)' && return 1
            return 0
        fi
        return 1
    fi
    # mariadb
    if [ -n "$MYSQL_HOST" ] && [ "$MYSQL_HOST" != "127.0.0.1" ] && [ "$MYSQL_HOST" != "localhost" ]; then
        return 0
    fi
    return 1
}

if skip_local_db; then
    echo "External DB configured — skipping local DB startup"
    cd "$CDIR"
    exec ./app
fi

# -----------------------------------------------------------------------
# All-in-one mode: fix permissions on volume-mounted dirs then start DB.
# chown/chmod g=u ensures an OpenShift-injected non-root UID in group 0
# can write to volume-mounted paths after a Velero restore.
# -----------------------------------------------------------------------

if [ "$DB_BACKEND" = "mariadb" ]; then
    # Fix volume permissions (Docker creates volumes as root:root)
    mkdir -p /var/lib/mysql/data /tmp/log/todoapp
    chown -R "$(id -u):0" /var/lib/mysql /tmp/log/todoapp 2>/dev/null || true
    chmod -R g=u /var/lib/mysql /tmp/log/todoapp 2>/dev/null || true

    # Delegate all DB initialisation to the official mariadb Docker entrypoint.
    # It reads MYSQL_ROOT_PASSWORD, MYSQL_USER, MYSQL_PASSWORD, MYSQL_DATABASE
    # and handles mariadb-install-db, socket setup, and user creation.
    # Try candidates in order: RHEL (run-mysqld), then upstream mariadb image.
    DB_PID=""
    for candidate in \
        "run-mysqld" \
        "/usr/local/bin/docker-entrypoint.sh mariadbd" \
        "/usr/local/bin/docker-entrypoint.sh mysqld" \
        "/docker-entrypoint.sh mariadbd" \
        "/docker-entrypoint.sh mysqld" \
        "mariadbd --user=$(id -u)" \
        "mysqld --user=$(id -u)"; do
        first="${candidate%% *}"
        if [[ "$first" == /* ]]; then
            [[ -x "$first" ]] || continue
        else
            command -v "$first" &>/dev/null || continue
        fi
        echo "Starting database with: $candidate"
        eval "$candidate" &
        DB_PID=$!
        break
    done

    if [ -z "$DB_PID" ]; then
        echo "ERROR: could not find a MariaDB start command" >&2
        exit 1
    fi

    echo "Waiting for MariaDB to accept connections (up to 120s)..."
    READY=0
    for i in $(seq 1 120); do
        # Wait until the app user is accessible — not just root ping —
        # so the Go app can connect immediately after exec.
        if mariadb -h 127.0.0.1 \
                -u "${MYSQL_USER:-changeme}" \
                -p"${MYSQL_PASSWORD:-changeme}" \
                "${MYSQL_DATABASE:-todolist}" \
                -e "SELECT 1" >/dev/null 2>&1; then
            echo "MariaDB is ready (user '${MYSQL_USER:-changeme}' can connect)"
            READY=1
            break
        fi
        if ! kill -0 "$DB_PID" 2>/dev/null; then
            echo "ERROR: MariaDB process exited unexpectedly" >&2
            exit 1
        fi
        sleep 1
    done

    if [ "$READY" -ne 1 ]; then
        echo "ERROR: MariaDB did not become ready in 120s" >&2
        exit 1
    fi

elif [ "$DB_BACKEND" = "mongodb" ]; then
    mkdir -p /var/lib/mongodb /tmp/log/todoapp
    chown -R "$(id -u):0" /var/lib/mongodb /tmp/log/todoapp 2>/dev/null || true
    chmod -R g=u /var/lib/mongodb /tmp/log/todoapp 2>/dev/null || true
    LOGPATH="/tmp/log/todoapp/mongod.log"

    echo "Starting MongoDB..."
    # Default in MongoDB 7 is no auth; --auth enables it. We omit it for local all-in-one.
    mongod --dbpath /var/lib/mongodb \
           --bind_ip 127.0.0.1 \
           --logpath "$LOGPATH" \
           --logappend \
           --fork 2>&1 || {
        echo "ERROR: mongod --fork failed" >&2
        exit 1
    }

    echo "Waiting for MongoDB to be ready (up to 120s)..."
    READY=0
    for i in $(seq 1 120); do
        # Use exit code; mongosh exits 0 only when the server responds.
        # Suppress output — newer mongosh prints { ok: 1 } (no quotes) so grep is unreliable.
        if mongosh --quiet --eval "db.adminCommand({ping:1})" >/dev/null 2>&1 || \
           mongo    --quiet --eval "db.adminCommand({ping:1})" >/dev/null 2>&1; then
            echo "MongoDB is ready"
            READY=1
            break
        fi
        sleep 1
    done

    if [ "$READY" -ne 1 ]; then
        echo "ERROR: MongoDB did not become ready in 120s" >&2
        exit 1
    fi
fi

cd "$CDIR"
exec ./app
