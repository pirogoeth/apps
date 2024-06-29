FROM golang:alpine AS build

COPY . /src
WORKDIR /src
RUN go build -o /nomad-deployer ./nomad-deployer

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates
COPY --from=build /nomad-deployer /nomad-deployer

ENTRYPOINT ["/nomad-deployer"]