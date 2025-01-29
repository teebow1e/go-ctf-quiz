# Build stage
FROM golang:latest AS build
WORKDIR /app

COPY go.* ./
RUN go mod download

RUN apt update
RUN apt install -y musl-tools

COPY . .
RUN CGO_ENABLED=1 CC=musl-gcc go build --ldflags '-linkmode=external -extldflags=-static' -o binary .

# Runtime stage
FROM gcr.io/distroless/base

COPY --from=build /app/binary /app/binary

CMD ["/app/binary"]
