FROM golang:1.24.4-alpine3.21 AS builder
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV TZ=Asia/Bangkok

WORKDIR /go/src

COPY go.mod .
COPY go.sum .
RUN go mod download

RUN apk update && apk upgrade && \
    apk add --no-cache tzdata ca-certificates git bash && \
    ln -snf "/usr/share/zoneinfo/$TZ" /etc/localtime && echo "$TZ" > /etc/timezone

COPY . .

RUN go build -o main .

FROM alpine:3.21
ENV TZ=Asia/Bangkok

RUN apk update && apk upgrade && \
    apk add --no-cache tzdata && \
    ln -snf /usr/share/zoneinfo/"$TZ" /etc/localtime && echo "$TZ" > /etc/timezone

COPY --from=builder /go/src/main /
EXPOSE 8082

CMD ["./main"]
