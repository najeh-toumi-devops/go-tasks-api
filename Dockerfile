# ---- Étape 1 : build ----
FROM golang:1.22-alpine AS build
WORKDIR /src
# On copie d'abord go.mod/go.sum pour profiter du cache Docker des dépendances
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO_ENABLED=0 => binaire statique, sans dépendance à la libc
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- Étape 2 : image finale minuscule (~10 Mo) ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
