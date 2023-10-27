package handler

import (
	"ecommerce-api/database"
	"ecommerce-api/helper"
	"ecommerce-api/model"
	"ecommerce-api/service"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	database    *database.Database
	validator   *helper.Validator
	authService *service.AuthService
}

func NewProductHandler(
	database *database.Database,
	validator *helper.Validator,
	authService *service.AuthService,
) *ProductHandler {
	return &ProductHandler{
		database:    database,
		validator:   validator,
		authService: authService,
	}
}

func (h *ProductHandler) GetAll(c echo.Context) error {
	products, err := model.GetAllProduct(h.database.Conn)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, products)
}

func (h *ProductHandler) GetByID(c echo.Context) error {
	product := model.Product{
		ID: c.Param("id"),
	}

	if err := product.GetByID(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) CreateCurrentStoreProduct(c echo.Context) error {
	price, err := strconv.ParseInt(c.FormValue("price"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid price")
	}

	stock, err := strconv.ParseInt(c.FormValue("stock"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid stock")
	}

	createRequest := model.ProductCreate{
		Name:        c.FormValue("name"),
		Description: c.FormValue("description"),
		Price:       price,
		Stock:       int(stock),
	}

	if err := h.validator.Validate(createRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	product := createRequest.ToProduct()

	store := model.Store{
		OwnerEmail: helper.ExtractJwtEmail(c),
	}
	if err := store.GetByOwnerEmail(h.database.Conn); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return echo.NewHTTPError(http.StatusBadRequest, "You don't have a store")
		}
		return echo.ErrInternalServerError
	}

	product.StoreID = store.ID

	if err := product.Create(h.database.Conn); err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) UpdateCurrentStoreProduct(c echo.Context) error {
	price, err := strconv.ParseInt(c.FormValue("price"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid price")
	}

	stock, err := strconv.ParseInt(c.FormValue("stock"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid stock")
	}

	updateRequest := model.ProductCreate{
		Name:        c.FormValue("name"),
		Description: c.FormValue("description"),
		Price:       price,
		Stock:       int(stock),
	}

	if err := h.validator.Validate(updateRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	product := updateRequest.ToProduct()
	product.ID = c.Param("id")
	if err := product.GetByID(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	store := model.Store{
		OwnerEmail: helper.ExtractJwtEmail(c),
	}
	if err := store.GetByOwnerEmail(h.database.Conn); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return echo.NewHTTPError(http.StatusBadRequest, "You don't have a store")
		}
		return echo.ErrInternalServerError
	}

	if product.StoreID != store.ID {
		return echo.NewHTTPError(http.StatusBadRequest, "You don't own this product")
	}

	if err := product.UpdateByID(h.database.Conn); err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) Buy(c echo.Context) error {
	quantity, err := strconv.ParseInt(c.FormValue("quantity"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid quantity")
	}

	transactionRequest := model.TransactionCreate{
		ProductID: c.Param("id"),
		Quantity:  int(quantity),
	}

	if err := h.validator.Validate(transactionRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.authService.CurrentUser(c)
	if err != nil {
		return echo.ErrUnauthorized
	}

	product := model.Product{
		ID: transactionRequest.ProductID,
	}

	if err := product.GetByID(h.database.Conn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	if product.Stock < transactionRequest.Quantity {
		return echo.NewHTTPError(http.StatusBadRequest, "Insufficient stock")
	}

	valueTransaction := product.Price * int64(transactionRequest.Quantity)

	if valueTransaction > user.Balance {
		return echo.NewHTTPError(http.StatusBadRequest, "Insufficient balance")
	}

	tx, err := h.database.Conn.Begin()
	if err != nil {
		return echo.ErrInternalServerError
	}

	user.Balance -= valueTransaction
	if err := user.Update(tx); err != nil {
		tx.Rollback()
		return echo.ErrInternalServerError
	}

	transaction := transactionRequest.ToTransaction()
	transaction.UserEmail = user.Email

	if err := transaction.Create(tx); err != nil {
		tx.Rollback()
		return echo.ErrInternalServerError
	}

	product.Stock -= transaction.Quantity
	if err := product.UpdateByID(tx); err != nil {
		tx.Rollback()
		return echo.ErrInternalServerError
	}

	if err := tx.Commit(); err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, transaction)
}
