FROM golang:alpine as builder

ADD main.go .
RUN CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o /fake-process .

FROM alpine:3.9
RUN apk upgrade --update --no-cache

USER nobody

COPY --from=builder /fake-process /usr/local/bin/

EXPOSE 8080

ENTRYPOINT [ "/usr/local/bin/fake-process" ]