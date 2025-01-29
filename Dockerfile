# Build stage
FROM golang:1.23 AS build
WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .
RUN go build -o binary .

# Runtime stage
FROM gcr.io/distroless/base
WORKDIR /app

COPY --from=build /app/binary /app/binary

ENTRYPOINT ["/app/binary"]
