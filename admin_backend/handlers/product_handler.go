package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/admin_backend/middleware"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ListProducts — админский листинг, включает неактивные и распроданные товары.
func (h *Handlers) ListProducts(c *gin.Context) {
	offset, limit := parsePagination(c)
	categoryID := parseOptionalIDQuery(c, "category_id")

	items, err := h.productService.ListAllAdmin(c.Request.Context(), offset, limit, categoryID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.productService.CountAllAdmin(c.Request.Context(), categoryID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.ProductAdminSummary]{Items: items, Total: total, Offset: offset, Limit: limit})
}

func (h *Handlers) GetProduct(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	product, err := h.productService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *Handlers) CreateProduct(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	var req dto.CreateProductRequest
	if !h.decodeJSON(c, &req) {
		return
	}
	price, err := models.NewMoneyFromFloat(req.Price)
	if err != nil {
		h.writeError(c, err)
		return
	}
	product, err := h.adminService.CreateProduct(c.Request.Context(), admin.TelegramID, req.CategoryID, req.Name, req.Description, price)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: product created", "admin_id", admin.TelegramID, "product_id", product.ID)
	c.JSON(http.StatusCreated, product)
}

func (h *Handlers) UpdateProduct(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateProductRequest
	if !h.decodeJSON(c, &req) {
		return
	}
	price, err := models.NewMoneyFromFloat(req.Price)
	if err != nil {
		h.writeError(c, err)
		return
	}
	product, err := h.adminService.UpdateProduct(c.Request.Context(), admin.TelegramID, id, req.CategoryID, req.Name, req.Description, price, req.IsActive)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: product updated", "admin_id", admin.TelegramID, "product_id", id)
	c.JSON(http.StatusOK, product)
}

func (h *Handlers) DeleteProduct(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.adminService.DeleteProduct(c.Request.Context(), admin.TelegramID, id); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: product deleted", "admin_id", admin.TelegramID, "product_id", id)
	c.Status(http.StatusNoContent)
}

func (h *Handlers) AddProductItems(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.AddItemsRequest
	if !h.decodeJSON(c, &req) {
		return
	}
	if err := h.adminService.AddProductItems(c.Request.Context(), admin.TelegramID, id, req.Contents); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: product items added", "admin_id", admin.TelegramID, "product_id", id, "count", len(req.Contents))
	c.Status(http.StatusNoContent)
}
