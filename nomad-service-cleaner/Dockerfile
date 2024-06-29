FROM golang:alpine AS build

COPY . /src
WORKDIR /src
RUN go build -o /nomad-service-cleaner ./nomad-service-cleaner

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates
COPY --from=build /nomad-service-cleaner /nomad-service-cleaner

ENTRYPOINT ["/nomad-service-cleaner"]