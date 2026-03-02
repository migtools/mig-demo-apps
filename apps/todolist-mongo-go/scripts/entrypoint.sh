#!/bin/bash
set -e

DATA_DIR="/data/db"
mkdir -p "$DATA_DIR"

MONGO_USER="${MONGO_INITDB_ROOT_USERNAME:-changeme}"
MONGO_PASS="${MONGO_INITDB_ROOT_PASSWORD:-changeme}"
MONGO_DB="${MONGO_INITDB_DATABASE:-todolist}"

# Detect first run: WiredTiger file is created by mongod on initialization
if [ ! -f "$DATA_DIR/WiredTiger" ]; then
    echo "First run detected — initializing MongoDB..."

    # Start mongod WITHOUT auth so we can create the root user
    mongod --dbpath "$DATA_DIR" --bind_ip 127.0.0.1 --fork --logpath /tmp/mongod-init.log

    # Wait for mongod to accept connections
    for i in $(seq 1 30); do
        if mongosh --quiet --eval "db.adminCommand('ping')" 2>/dev/null; then
            echo "mongod (no-auth) is ready"
            break
        fi
        sleep 1
    done

    # Create the root/admin user
    mongosh admin --quiet --eval "
        db.createUser({
            user: '$MONGO_USER',
            pwd:  '$MONGO_PASS',
            roles: [{ role: 'root', db: 'admin' }]
        });
    "
    echo "Created admin user '$MONGO_USER'"

    # Shut down the temporary no-auth instance.
    # The shutdown causes mongosh to lose its connection, which returns a
    # non-zero exit code — this is expected, so we suppress the error.
    mongosh admin --quiet --eval "db.shutdownServer()" 2>/dev/null || true
    sleep 2
    echo "MongoDB initialization complete"
fi

# Start mongod WITH auth in background
mongod --auth --dbpath "$DATA_DIR" --bind_ip 127.0.0.1 &
MONGO_PID=$!

# Wait for MongoDB to accept authenticated connections (up to 60 seconds)
echo "Waiting for MongoDB to start..."
for i in $(seq 1 60); do
    if mongosh admin -u "$MONGO_USER" -p "$MONGO_PASS" --quiet --eval "db.adminCommand('ping')" 2>/dev/null; then
        echo "MongoDB is ready"
        break
    fi
    if ! kill -0 $MONGO_PID 2>/dev/null; then
        echo "MongoDB process died unexpectedly"
        exit 1
    fi
    sleep 1
done

# Start the todolist Go application (foreground, replaces shell)
cd /opt/todolist
exec ./app
