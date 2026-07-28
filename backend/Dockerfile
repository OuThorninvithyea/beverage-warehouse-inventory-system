FROM golang:1.26.1-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/bwims-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/bwims-seed ./cmd/seed

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
USER app
COPY --from=build /bin/bwims-api /usr/local/bin/bwims-api
COPY --from=build /bin/bwims-seed /usr/local/bin/bwims-seed
EXPOSE 8080
ENTRYPOINT ["bwims-api"]
