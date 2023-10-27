package service

import (
	"ecommerce-api/database"
	"ecommerce-api/helper"
	"ecommerce-api/model"

	"github.com/labstack/echo/v4"
)

type AuthService struct {
	database *database.Database
}

func NewAuthService(database *database.Database) *AuthService {
	return &AuthService{
		database: database,
	}
}

func (s *AuthService) CurrentUser(c echo.Context) (model.User, error) {
	user := model.User{
		Email: helper.ExtractJwtEmail(c),
	}

	if err := user.GetByEmail(s.database.Conn); err != nil {
		return user, err
	}

	return user, nil
}
