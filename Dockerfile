# syntax=docker/dockerfile:1

FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/reactorlog ./cmd/reactorlog

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/reactorlog /app/reactorlog
# CI copies web/dist into static/ before docker build.
COPY static /app/static
ENV STATIC_DIR=/app/static
ENV HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/reactorlog"]
CMD ["server"]
