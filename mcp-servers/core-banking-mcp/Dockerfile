FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY fixtures/ ./fixtures/
COPY mcp-servers/core-banking-mcp/ ./mcp-servers/core-banking-mcp/
WORKDIR /src/mcp-servers/core-banking-mcp
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/app /app
EXPOSE 9110
ENTRYPOINT ["/app"]
