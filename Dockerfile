# syntax=docker/dockerfile:1
# Works with Docker and Apple `container` (macOS 26+).

# ---- frontend ----
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- backend (embeds web/dist) ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/siloam .

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/siloam /siloam
ENV ADDR=:8080 \
    MONGODB_URI=mongodb://mongo:27017/siloam
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/siloam"]
