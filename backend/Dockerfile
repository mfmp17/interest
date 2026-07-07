FROM golang:1.26-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/fred-backend .

FROM alpine:3.21
WORKDIR /app
RUN mkdir -p /data
COPY --from=build /out/fred-backend /app/fred-backend
ENV PORT=8090
ENV FRED_STATE_PATH=/data/mainnet-state.json
EXPOSE 8090
CMD ["/app/fred-backend"]
