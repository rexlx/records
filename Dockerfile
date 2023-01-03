FROM golang:1.19-alpine AS builder
WORKDIR /app
ADD . /app
RUN apk update && apk add build-base
RUN cd /app && CGO_ENABLED=0 go build -ldflags="-w -s" -o main ./source/main/

# FROM alpine
# WORKDIR /app
# COPY --from=builder /app/main /app/new-main
# COPY --from=builder /app/config.json /app/config.json
CMD ["/app/main"]