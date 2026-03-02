#!/bin/bash
set -e

# When MYSQL_HOST points to a remote host (e.g. "mysql" in docker-compose), we are
# not running MariaDB in this container; only run the Go app.
if [ -n "$MYSQL_HOST" ] && [ "$MYSQL_HOST" != "127.0.0.1" ] && [ "$MYSQL_HOST" != "localhost" ]; then
    cd /opt/todolist
    exec ./app
fi

# Start MariaDB/MySQL in background. Try multiple commands so we work across images:
#   run-mysqld       - RHEL/Red Hat mariadb image
#   docker-entrypoint.sh + mariadbd/mysqld - official MariaDB/MySQL Docker image
#   mariadbd / mysqld - direct binary (many distros)
MYSQL_PID=""
for candidate in \
    "run-mysqld" \
    "/usr/local/bin/docker-entrypoint.sh mariadbd" \
    "/usr/local/bin/docker-entrypoint.sh mysqld" \
    "/docker-entrypoint.sh mariadbd" \
    "/docker-entrypoint.sh mysqld" \
    "mariadbd" \
    "mysqld"; do
    first="${candidate%% *}"
    if [[ "$first" == /* ]]; then
        [[ -x "$first" ]] || continue
    else
        command -v "$first" &>/dev/null || continue
    fi
    echo "Starting database with: $candidate"
    eval "$candidate" &
    MYSQL_PID=$!
    break
done

if [ -z "$MYSQL_PID" ]; then
    echo "Could not find any MariaDB/MySQL start command (tried run-mysqld, docker-entrypoint.sh, mariadbd, mysqld)"
    exit 1
fi

# Wait for MariaDB to accept connections (up to 60 seconds)
echo "Waiting for MariaDB to start..."
for i in $(seq 1 60); do
    if mysqladmin ping -h 127.0.0.1 --silent 2>/dev/null; then
        echo "MariaDB is ready"
        break
    fi
    if ! kill -0 $MYSQL_PID 2>/dev/null; then
        echo "MariaDB process died unexpectedly"
        exit 1
    fi
    sleep 1
done

# Start the todolist Go application (foreground, replaces shell)
cd /opt/todolist
exec ./app
