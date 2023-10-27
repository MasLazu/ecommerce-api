package handler

import (
	"ecommerce-api/database"
	"ecommerce-api/helper"
	"ecommerce-api/model"
	"ecommerce-api/service"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	database    *database.Database
	validator   *helper.Validator
	authService *service.AuthService
}

func NewUserHandler(database *database.Database, validator *helper.Validator, authService *service.AuthService) *UserHandler {
	return &UserHandler{
		database:    database,
		validator:   validator,
		authService: authService,
	}
}

func (h *UserHandler) Register(c echo.Context) error {
	registerRequest := model.UserRegister{
		Email:     c.FormValue("email"),
		FirstName: c.FormValue("first_name"),
		LastName:  c.FormValue("last_name"),
		Password:  c.FormValue("password"),
	}

	if err := h.validator.Validate(registerRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user := registerRequest.ToUser()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.ErrInternalServerError
	}

	user.Password = string(hashedPassword)

	if err := user.Create(h.database.Conn); err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
			return echo.NewHTTPError(http.StatusBadRequest, "Email already taken")
		}
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetByEmail(c echo.Context) error {
	user := model.User{
		Email: c.Param("email"),
	}

	if err := user.GetByEmail(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetAll(c echo.Context) error {
	users, err := model.GetAllUsers(h.database.Conn)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, users)
}

func (h *UserHandler) GetCurrent(c echo.Context) error {
	user, err := h.authService.CurrentUser(c)
	if err != nil {
		return echo.ErrUnauthorized
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateCurrent(c echo.Context) error {
	updateRequest := model.UserUpdate{
		FirstName: c.FormValue("first_name"),
		LastName:  c.FormValue("last_name"),
	}

	if err := h.validator.Validate(updateRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user := updateRequest.ToUser()
	user.Email = helper.ExtractJwtEmail(c)
	user.Update(h.database.Conn)

	return c.JSON(http.StatusOK, user)
}
