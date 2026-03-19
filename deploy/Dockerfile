# Build stage
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app
COPY backend/ ./backend/
RUN cd backend && go mod tidy && go build -o homepagex

# Frontend is static, no build needed

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy backend binary
COPY --from=backend-builder /app/backend/homepagex ./

# Copy frontend static files
COPY frontend/build/ ./frontend/build/

# Copy config and pages
COPY config.yaml ./
COPY pages/ ./pages/

# Expose port
EXPOSE 8090

# Run
CMD ["./homepagex"]
