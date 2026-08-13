FROM golang:1.26-trixie AS build

# Supplied by the release workflow. Without them a build reports itself as dev,
# which is what a build outside a release is.
ARG VERSION=dev
ARG COMMIT=none

WORKDIR /src
COPY go.mod *.go ./
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /github-app-tokend

FROM debian:trixie-slim

# The one request this program makes goes to GitHub over TLS, and the slim
# image ships no CA bundle to verify it against.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --no-create-home --uid 10001 --user-group github-app-tokend

COPY --from=build /github-app-tokend /usr/local/bin/github-app-tokend

USER github-app-tokend

ENTRYPOINT ["/usr/local/bin/github-app-tokend"]
CMD ["-h"]
