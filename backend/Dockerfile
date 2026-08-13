FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /rndops-api ./cmd/api

FROM alpine:3.19
COPY --from=build /rndops-api /rndops-api
EXPOSE 8080
ENTRYPOINT ["/rndops-api"]
