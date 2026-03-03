#!/bin/bash
set -e

# Start MariaDB in background using the image's built-in run-mysqld.
# run-mysqld handles data dir initialization and user/database creation
# from env vars (MYSQL_USER, MYSQL_PASSWORD, MYSQL_ROOT_PASSWORD, MYSQL_DATABASE).
run-mysqld &
MYSQL_PID=$!

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
