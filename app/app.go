package app

import (
	"ecommerce-api/database"
	"ecommerce-api/handler"
	"ecommerce-api/helper"
	"ecommerce-api/middleware"
	"ecommerce-api/service"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type App struct {
	Instance *echo.Echo
	Config   *Config
	Database *database.Database
}

func NewApp() *App {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	config := NewConfig()
	database := database.NewDatabase(config.DatabaseUrl)
	validator := helper.NewValidator()
	authService := service.NewAuthService(database)
	authHandler := handler.NewAuthHandler(database, validator, config.Jwt.SigningKey.([]byte))
	userHandler := handler.NewUserHandler(database, validator, authService)
	storeHandler := handler.NewStoreHandler(database, validator)
	productHandler := handler.NewProductHandler(database, validator, authService)
	transactionHandler := handler.NewTransactionHandler(database, validator)
	authMiddleware := middleware.NewAuthMiddleware(config.Jwt)
	instance := echo.New()
	SetupRoute(instance, authHandler, userHandler, storeHandler, productHandler, transactionHandler, authMiddleware)

	return &App{
		Instance: instance,
		Config:   config,
		Database: database,
	}
}

func (app *App) Start() {
	app.Instance.Logger.Fatal(app.Instance.Start(":" + app.Config.Port))
}
