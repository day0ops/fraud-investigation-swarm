FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY fixtures/ ./fixtures/
COPY mcp-servers/typology-mcp/ ./mcp-servers/typology-mcp/
WORKDIR /src/mcp-servers/typology-mcp
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/app /app
EXPOSE 9113
ENTRYPOINT ["/app"]
