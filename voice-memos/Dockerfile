FROM golang:alpine AS build

COPY . /src
WORKDIR /src
RUN go build -o /voice-memos ./voice-memos

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates
COPY --from=build /voice-memos /voice-memos

ENTRYPOINT ["/voice-memos"]