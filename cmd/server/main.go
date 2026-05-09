package main

import (
	"ax-management/internal/api"
	"ax-management/internal/config"
	models "ax-management/internal/model"
	"ax-management/internal/repository"
	"ax-management/internal/service"
	"log"

	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. Load Configuration
	cfg := config.GetConfig()

	// 2. Database Initialization
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Critical: Could not connect to Postgres: %v", err)
	}

	// Auto-Migrate (Enterprise Tip: In production, use 'golang-migrate' instead)
	db.AutoMigrate(&models.User{}, &models.Tenant{}, &models.ApiKey{})

	// 3. Cache Initialization
	rdb, err := repository.NewRedisClient(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Critical: Could not connect to Redis: %v", err)
	}

	isProd := cfg.Environment == "production"

	repo := repository.NewPostgresRepository(db)
	mgtSvc := service.NewManagementService(repo, rdb)
	authSvc := service.NewAuthService(db, cfg.JWTSecret) // Inject Secret
	hdl := api.NewHandler(mgtSvc, authSvc, isProd)

	// 4. Dependency Injection
	//svc := service.NewManagementService(repo, rdb)

	// 5. App Setup
	app := fiber.New(fiber.Config{
		AppName: "AX Management Control Plane",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowHeaders:     "Origin, Content-Type, Accept, X-API-Key",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	api.SetupRoutes(app, hdl)

	// 6. Graceful Startup
	log.Printf("Management API starting on port %s in %s mode", cfg.AppPort, cfg.Environment)
	log.Fatal(app.Listen(":" + cfg.AppPort))
}
