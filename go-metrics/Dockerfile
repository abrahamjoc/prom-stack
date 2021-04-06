FROM golang:alpine AS builder
WORKDIR $GOPATH/src/go-metrics
COPY . .
RUN go build -o /go/bin/app

FROM golang:alpine
COPY --from=builder /go/bin/app /bin/go-metrics
ENTRYPOINT ["go-metrics"]