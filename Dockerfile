# build stage
FROM golang as builder

ENV GO111MODULE=on

WORKDIR /app

COPY *.go go.mod go.sum /app/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor




FROM alpine:3.7
RUN apk add --update sqlite

COPY --from=builder /app/superman-detector /app/
COPY ./GeoLite2-City-Blocks-IPv4.db /db/

CMD ["/app/superman-detector"]
