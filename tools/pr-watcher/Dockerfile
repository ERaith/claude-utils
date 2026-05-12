# Multi-stage build for pr-watcher.
#
# NOTE: The runtime image needs the `gh`, `glab`, and `claude` CLIs to be useful.
# The default base image below installs gh + glab; `claude` must be mounted into
# the container at runtime (or installed in a derived image) because it requires
# an interactive login the first time. The container is intended for a host
# where you can mount ~/.config/{gh,glab,claude}.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/pr-watcher .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates github-cli glab tini
COPY --from=build /out/pr-watcher /usr/local/bin/pr-watcher
ENV LOG_LEVEL=info
EXPOSE 8782
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/pr-watcher"]
CMD ["--config", "/etc/pr-watcher/config.yaml"]
