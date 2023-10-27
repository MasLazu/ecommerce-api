package handler

import (
	"ecommerce-api/database"
	"ecommerce-api/helper"
	"ecommerce-api/model"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	database  *database.Database
	validator *helper.Validator
	jwtKey    []byte
}

func NewAuthHandler(
	database *database.Database,
	validator *helper.Validator,
	jwtKey []byte,
) *AuthHandler {
	return &AuthHandler{
		database:  database,
		validator: validator,
		jwtKey:    jwtKey,
	}
}

func (h *AuthHandler) Login(c echo.Context) error {
	loginRequest := model.UserLogin{
		Email:    c.FormValue("email"),
		Password: c.FormValue("password"),
	}

	if err := h.validator.Validate(loginRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	password := loginRequest.Password
	user := loginRequest.ToUser()
	if err := user.GetByEmail(h.database.Conn); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid email or password")
		}
		return echo.ErrInternalServerError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid email or password")
	}

	store := model.Store{
		OwnerEmail: user.Email,
	}
	if err := store.GetByOwnerEmail(h.database.Conn); err != nil {
		if err.Error() == "sql: no rows in result set" {
			store.ID = ""
		} else {
			return echo.ErrInternalServerError
		}
	}

	refreshToken := model.RefreshToken{
		UserEmail: user.Email,
	}
	if err := refreshToken.Create(h.database.Conn); err != nil {
		return echo.ErrInternalServerError
	}

	helper.AssignRefreshTokenCookes(refreshToken.Token, c)

	signedToken, err := helper.GenerateJwtToken(user.Email, h.jwtKey)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, echo.Map{
		"access_token": signedToken,
	})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid refresh token")
	}

	refreshToken := model.RefreshToken{
		Token: cookie.Value,
	}

	store := model.Store{
		OwnerEmail: refreshToken.UserEmail,
	}
	if err := store.GetByOwnerEmail(h.database.Conn); err != nil {
		if err.Error() == "sql: no rows in result set" {
			store.ID = ""
		} else {
			return echo.ErrInternalServerError
		}
	}

	if err := refreshToken.GetByToken(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid refresh token")
	}

	signedToken, err := helper.GenerateJwtToken(refreshToken.UserEmail, h.jwtKey)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, echo.Map{
		"access_token": signedToken,
	})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid refresh token")
	}

	refreshToken := model.RefreshToken{
		Token: cookie.Value,
	}

	if err := refreshToken.Delete(h.database.Conn); err != nil {
		return echo.ErrInternalServerError
	}

	helper.ClearRefreshTokenCookies(c)

	return c.NoContent(http.StatusOK)
}
