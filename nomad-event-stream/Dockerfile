FROM golang:alpine AS build

COPY . /src
WORKDIR /src
RUN go build -o /nomad-event-stream ./nomad-event-stream/...

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates
COPY --from=build /nomad-event-stream /nomad-event-stream

ENTRYPOINT ["/nomad-event-stream"]