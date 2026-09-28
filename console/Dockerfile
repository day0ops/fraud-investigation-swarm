FROM node:22-alpine AS web
WORKDIR /web
COPY console/web/package.json console/web/package-lock.json* ./
RUN npm install
COPY console/web/ ./
RUN npm run build

FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY fixtures/ ./fixtures/
COPY console/ ./console/
COPY --from=web /web/dist ./console/web/dist
WORKDIR /src/console
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app .

# kubectl-ate is the substrate CLI the backend shells out to for live actor
# state (see console/source.go); it is a statically-linked Go binary itself,
# so it runs fine under distroless static-debian12 below.
FROM alpine:3.20 AS ate-fetch
RUN apk add --no-cache curl
ARG KUBECTL_ATE_VERSION=v0.2.0-beta5
RUN curl -fsSL -o /usr/local/bin/kubectl-ate "https://github.com/kagent-dev/substrate/releases/download/${KUBECTL_ATE_VERSION}/kubectl-ate-linux-amd64" \
    && chmod +x /usr/local/bin/kubectl-ate

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/app /app
COPY --from=ate-fetch /usr/local/bin/kubectl-ate /usr/local/bin/kubectl-ate
EXPOSE 8090
ENTRYPOINT ["/app"]
