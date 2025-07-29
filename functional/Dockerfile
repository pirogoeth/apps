FROM docker.io/library/golang:alpine AS build

ENV CGO_ENABLED=1

WORKDIR /src
ADD . /src
WORKDIR /src/functional
RUN mkdir /app \
    && apk add \
        build-base \
        sqlite-libs \
    && go build -o /app/functional ./

# ---

FROM docker.io/library/alpine:latest

WORKDIR /app
RUN apk add sqlite-libs
COPY --from=build /app/functional /app/functional

VOLUME /config
VOLUME /data

ENV GIN_MODE=release

ENTRYPOINT ["/app/functional"]
