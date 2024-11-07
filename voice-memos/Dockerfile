FROM golang:alpine AS build

RUN apk --no-cache add file-dev gcc musl-dev

COPY . /src
WORKDIR /src
RUN go build -o /voice-memos ./voice-memos

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates file mailcap

COPY --from=build /voice-memos /voice-memos

ENTRYPOINT ["/voice-memos"]