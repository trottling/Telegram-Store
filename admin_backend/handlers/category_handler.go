package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/admin_backend/middleware"
)

// ListCategories отдаёт плоский список всех категорий — дерево строит фронтенд.
func (h *Handlers) ListCategories(c *gin.Context) {
	categories, err := h.categoryService.ListAllFlat(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *Handlers) GetCategory(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	category, err := h.categoryService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, category)
}

func (h *Handlers) CreateCategory(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	var req dto.CreateCategoryRequest
	if !h.decodeJSON(c, &req) {
		return
	}
	category, err := h.adminService.CreateCategory(c.Request.Context(), admin.TelegramID, req.ParentID, req.Name, req.Description)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: category created", "admin_id", admin.TelegramID, "category_id", category.ID)
	c.JSON(http.StatusCreated, category)
}

func (h *Handlers) UpdateCategory(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCategoryRequest
	if !h.decodeJSON(c, &req) {
		return
	}
	category, err := h.adminService.UpdateCategory(c.Request.Context(), admin.TelegramID, id, req.Name, req.Description, req.ParentID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: category updated", "admin_id", admin.TelegramID, "category_id", id)
	c.JSON(http.StatusOK, category)
}

func (h *Handlers) DeleteCategory(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.adminService.DeleteCategory(c.Request.Context(), admin.TelegramID, id); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: category deleted", "admin_id", admin.TelegramID, "category_id", id)
	c.Status(http.StatusNoContent)
}
