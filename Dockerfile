FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /goproxy .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /goproxy /goproxy
COPY config.yaml /config.yaml
EXPOSE 9090
CMD ["/goproxy"]
