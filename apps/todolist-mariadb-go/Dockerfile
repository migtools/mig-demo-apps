# Stage 1: Build Go binary
FROM golang:1.24-alpine AS build-env
WORKDIR /build
COPY *.go go.mod go.sum ./
RUN go mod download && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o app

# Stage 2: Combined MariaDB + todolist container
# rhel9/mariadb-1011 is built on UBI-9 and has MariaDB 10.11 pre-installed.
# It handles DB initialization automatically via env vars:
#   MYSQL_USER, MYSQL_PASSWORD, MYSQL_ROOT_PASSWORD, MYSQL_DATABASE
FROM registry.redhat.io/rhel9/mariadb-1011

USER root

# Copy Go binary and web assets
COPY --from=build-env /build/app /opt/todolist/app
COPY resources/ /opt/todolist/resources/
COPY index.html /opt/todolist/
COPY scripts/entrypoint.sh /opt/todolist/entrypoint.sh
RUN chmod +x /opt/todolist/entrypoint.sh

EXPOSE 8000
CMD ["/opt/todolist/entrypoint.sh"]
