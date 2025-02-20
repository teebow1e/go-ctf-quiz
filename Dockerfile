# Build stage
FROM golang:1.23 AS build
WORKDIR /app

# ENV GOPROXY=https://goproxy.cn,direct
# ENV GOPROXY=direct

COPY go.* ./
RUN go mod download

COPY . .
RUN go build -o binary .

# Runtime stage
FROM gcr.io/distroless/base
WORKDIR /app

COPY --from=build /app/binary /app/binary

ENTRYPOINT ["/app/binary"]
