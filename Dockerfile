FROM golang:1.14 AS builder

COPY . /mediasync-server

WORKDIR /mediasync-server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o mediasync-server main.go

FROM scratch

WORKDIR /

COPY --from=builder /mediasync-server/mediasync-server /

ENV PORT=4242
EXPOSE 4242

CMD ["/mediasync-server"]
