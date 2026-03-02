# Stage 1: Build Go binary
FROM golang:1.24-alpine AS build-env
WORKDIR /build
COPY *.go go.mod go.sum ./
RUN go mod download && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o app

# Stage 2: Combined MongoDB + todolist container on UBI 9
# We install MongoDB 7.0 Community from the official yum repo since
# Red Hat does not provide a pre-built MongoDB UBI image (unlike MariaDB).
FROM registry.redhat.io/ubi9/ubi

USER root

# Install MongoDB 7.0 from official repo
COPY mongodb-org-7.0.repo /etc/yum.repos.d/mongodb-org-7.0.repo
RUN dnf install -y mongodb-org && dnf clean all

# Copy Go binary and web assets
COPY --from=build-env /build/app /opt/todolist/app
COPY resources/ /opt/todolist/resources/
COPY index.html /opt/todolist/
COPY favicon.ico /opt/todolist/
COPY scripts/entrypoint.sh /opt/todolist/entrypoint.sh
RUN chmod +x /opt/todolist/entrypoint.sh

# Create MongoDB data directory
RUN mkdir -p /data/db

EXPOSE 8000
CMD ["/opt/todolist/entrypoint.sh"]
