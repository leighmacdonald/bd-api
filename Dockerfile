FROM golang:1.22-alpine as build
WORKDIR /build
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -gcflags="all=-N -l"

FROM alpine:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/leighmacdonald/bd-api"
EXPOSE 8890 40000
RUN apk add dumb-init
WORKDIR /app
COPY --from=build /go/bin/dlv /app/
COPY --from=build /build/bd-api /app/
ENTRYPOINT ["dumb-init", "--"]
CMD ["./bd-api", "run"]
CMD ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/server"]