# Build from the repository root: docker build -f services/indexer/Dockerfile .
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY services ./services
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/indexer ./services/indexer/cmd/indexer

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/indexer /indexer
ENTRYPOINT ["/indexer"]
