# Build from the repository root: docker build -f services/worker/Dockerfile .
# The Engine is built from the pinned commit that the worker integration was
# verified against (see docs/engine-integration.md).
FROM rust:1.99-bookworm AS engine
ARG ENGINE_REF=78b385fd78e7dde034145f680a15f21e7071eda4
RUN git init /engine \
 && git -C /engine fetch --depth 1 https://github.com/UpgradeRail/upgraderail-engine.git ${ENGINE_REF} \
 && git -C /engine checkout --detach FETCH_HEAD
WORKDIR /engine
RUN cargo build --release --locked -p upgraderail-cli \
 && strip target/release/upgraderail

FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY services ./services
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./services/worker/cmd/worker

FROM debian:bookworm-slim
# CA certificates are required for HTTPS to the S3-compatible artifact store.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --system --uid 10001 --create-home worker
COPY --from=engine /engine/target/release/upgraderail /usr/local/bin/upgraderail
COPY --from=build /out/worker /usr/local/bin/worker
ENV UPGRADERAIL_ENGINE_BIN=/usr/local/bin/upgraderail \
    UPGRADERAIL_WORK_DIR=/tmp/upgraderail-work
USER worker
ENTRYPOINT ["/usr/local/bin/worker"]
