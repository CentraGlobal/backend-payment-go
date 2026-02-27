package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// HealthHandler returns an extended health-check handler that verifies DB and
// Redis connectivity in addition to reporting the service as alive.
func HealthHandler(db *pgxpool.Pool, ariDB *pgxpool.Pool, redisClient *goredis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		status := fiber.Map{
			"status": "healthy",
		}

		if db != nil {
			if err := db.Ping(ctx); err != nil {
				status["database"] = "unhealthy"
				status["status"] = "degraded"
			} else {
				status["database"] = "healthy"
			}
		}

		if ariDB != nil {
			if err := ariDB.Ping(ctx); err != nil {
				status["ari_database"] = "unhealthy"
				status["status"] = "degraded"
			} else {
				status["ari_database"] = "healthy"
			}
		}

		if redisClient != nil {
			if err := redisClient.Ping(ctx).Err(); err != nil {
				status["redis"] = "unhealthy"
				status["status"] = "degraded"
			} else {
				status["redis"] = "healthy"
			}
		}

		httpStatus := fiber.StatusOK
		if status["status"] == "degraded" {
			httpStatus = fiber.StatusServiceUnavailable
		}

		return c.Status(httpStatus).JSON(status)
	}
}
