FROM golang:alpine AS build

COPY . /src
WORKDIR /src
RUN go build -o /nomad-external-dns ./nomad-external-dns/...

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates
COPY --from=build /nomad-external-dns /nomad-external-dns

ENTRYPOINT ["/nomad-external-dns"]