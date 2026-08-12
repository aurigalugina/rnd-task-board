FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /myagenda-service ./cmd/api

FROM alpine:3.19
COPY --from=build /myagenda-service /myagenda-service
EXPOSE 8090
ENTRYPOINT ["/myagenda-service"]
