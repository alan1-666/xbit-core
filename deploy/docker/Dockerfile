FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

ARG SERVICE=gateway
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/${SERVICE}

FROM alpine:3.21
RUN addgroup -S xbit && adduser -S xbit -G xbit
USER xbit
COPY --from=build /out/server /server
ENTRYPOINT ["/server"]
