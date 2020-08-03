FROM golang:1.14 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o operator ./cmd/operator

FROM gcr.io/distroless/static
WORKDIR /root
COPY --from=builder /src/operator .
ENTRYPOINT [ "./operator", "-cli"]
