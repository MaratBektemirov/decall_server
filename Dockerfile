FROM golang:1.24-bookworm AS development
WORKDIR /app
ENV CGO_ENABLED=0
RUN go install github.com/air-verse/air@v1.52.3 \
	&& install -m755 /go/bin/air /usr/local/bin/air
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server /app/server
USER nobody:nobody
EXPOSE 8080
ENTRYPOINT ["/app/server"]
