// Package forgekit is a Go CLI tool that generates production-ready REST APIs
// with hexagonal architecture.
//
// ForgeKit scaffolds Go backend projects with:
//   - Hexagonal architecture (domain, application, infrastructure, transport)
//   - PostgreSQL with database/sql (no ORM)
//   - Chi router (github.com/go-chi/chi/v5)
//   - Docker & Docker Compose for local development
//   - Database migrations
//   - Environment configuration via .env (caarlos0/env pattern)
//   - Table-driven tests
//   - Architecture validation (check, analyze, doctor commands)
//   - Extensible feature system (forge add auth, cors, logging, swagger)
//
// Installation:
//
//	go install github.com/Demetrius-ch/forgekit/cmd/forge@latest
//
// Quick start:
//
//	forge init my-api
//	cd my-api
//	cp .env.example .env
//	docker compose -f docker/docker-compose.yml up --build
//
// Documentation: https://github.com/Demetrius-ch/forgekit
package forgekit
