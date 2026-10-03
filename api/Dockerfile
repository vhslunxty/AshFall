FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go mod tidy && CGO_ENABLED=0 go build -o /api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /api /api
EXPOSE 8080
CMD ["/api"]
