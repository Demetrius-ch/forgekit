package application

import "github.com/test/test-api/internal/domain"

// Health returns the service health without depending on transport or infrastructure.
func Health() domain.Health {
	return domain.Health{Status: "ok"}
}
