FROM golang:1.23.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api .

FROM alpine:3.20.3
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/api /usr/local/bin/api
USER app

EXPOSE 8080
ENTRYPOINT ["api"]
