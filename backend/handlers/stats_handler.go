package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Dashboard — данные экрана статистики, ?from/?to сужают период.
func (h *Handlers) Dashboard(c *gin.Context) {
	from := parseOptionalTimeQuery(c, "from")
	to := parseOptionalTimeQuery(c, "to")

	dashboard, err := h.statsService.GetDashboard(c.Request.Context(), from, to)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}
