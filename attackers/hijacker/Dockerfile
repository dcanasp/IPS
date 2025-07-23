FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY main.go .

# Build a statically linked Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix static -o attacker main.go

# Stage 2: Use distroless image for final runtime
FROM gcr.io/distroless/static-debian11

COPY --from=builder /app/attacker /

USER nonroot:nonroot

ENTRYPOINT ["/attacker"]
