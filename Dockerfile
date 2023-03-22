FROM golang:1.20-alpine as build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build

FROM alpine:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/leighmacdonald/bd-api"
EXPOSE 8888
ENV STEAM_API_KEY ""
RUN apk add dumb-init
WORKDIR /app
COPY --from=build /build/bd-api .
ENTRYPOINT ["dumb-init", "--"]
VOLUME /app/.cache
CMD ["./bd-api"]
