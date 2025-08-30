#
# Build go project
#
FROM golang:1.24-alpine as go-builder

WORKDIR /database-controller

COPY . .

RUN apk add --no-cache -t build-tools curl git && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o database-controller cmd/controller/*.go && \
    apk del build-tools && \
    rm -rf /var/cache/apk/*

#
# Runtime container
#
FROM alpine:latest  

WORKDIR /app

RUN apk --no-cache add ca-certificates bash curl

COPY --from=go-builder /database-controller/database-controller /database-controller

ENTRYPOINT ["/database-controller"]