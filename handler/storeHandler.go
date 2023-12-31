package handler

import (
	"ecommerce-api/database"
	"ecommerce-api/helper"
	"ecommerce-api/model"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type StoreHandler struct {
	database  *database.Database
	validator *validator.Validate
}

func NewStoreHandler(database *database.Database, validator *validator.Validate) *StoreHandler {
	return &StoreHandler{
		database:  database,
		validator: validator,
	}
}

func (h *StoreHandler) GetAll(c echo.Context) error {
	stores, err := model.GetAllStore(h.database.Conn)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, stores)
}

func (h *StoreHandler) GetByID(c echo.Context) error {
	store := model.Store{
		ID: c.Param("id"),
	}

	if err := store.GetByID(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Store not found")
	}

	return c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) CreateCurrentUserStore(c echo.Context) error {
	registerRequest := model.StoreRegister{
		Name: c.FormValue("name"),
	}

	if err := h.validator.Struct(registerRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	store := registerRequest.ToStore()
	store.OwnerEmail = helper.ExtractJwtEmail(c)

	if err := store.Create(h.database.Conn); err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
			return echo.NewHTTPError(http.StatusBadRequest, "This account already has a store")
		}
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, store)
}

func (h *StoreHandler) GetCurrent(c echo.Context) error {
	store := model.Store{
		OwnerEmail: helper.ExtractJwtEmail(c),
	}

	if err := store.GetByOwnerEmail(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Store not found")
	}

	return c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) UpdateCurrent(c echo.Context) error {
	updateRequest := model.StoreRegister{
		Name: c.FormValue("name"),
	}

	if err := h.validator.Struct(updateRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	store := updateRequest.ToStore()
	store.OwnerEmail = helper.ExtractJwtEmail(c)

	if err := store.UpdateByOwnerEmail(h.database.Conn); err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, store)
}
