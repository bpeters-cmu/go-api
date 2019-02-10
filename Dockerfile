# build stage
FROM golang as builder

ENV GO111MODULE=on

WORKDIR $GOPATH/src/superman-detector
COPY . .


RUN CGO_ENABLED=1 GOOS=linux go build -mod vendor

# run stage
FROM alpine:3.7
RUN apk add --update sqlite gcc musl-dev libc6-compat

WORKDIR /app/
COPY --from=builder /go/src/superman-detector/superman-detector .
COPY GeoLite2-City-Blocks-IPv4.db .
EXPOSE 3000
CMD ["./superman-detector"]
