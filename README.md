# bff-api-go-collection-activity-log

Go BFF para el módulo de activity-log de collections.

## Stack

- Go 1.22+
- Gin (HTTP)
- OpenTelemetry (observability)
- mTLS BFF→Core

## Estructura

```
cmd/server/main.go
internal/
  application/usecases/
  domain/
  infrastructure/
  interface/api/
Dockerfile
Makefile
go.mod
```
