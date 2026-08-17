package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/admin_backend/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	maxBodyBytes     = 1 << 20
)

// writeError логирует ошибку (5xx — Error, 4xx — Debug) и пишет HTTP-ответ.
func (h *Handlers) writeError(c *gin.Context, err error) {
	status, body := errors.DomainErrorToResponse(err)
	fields := logrus.Fields{"method": c.Request.Method, "path": c.Request.URL.Path, "status": status}
	if status >= http.StatusInternalServerError {
		h.log.WithError(err).WithFields(fields).Error("handlers: request failed")
	} else {
		h.log.WithError(err).WithFields(fields).Debug("handlers: request rejected")
	}
	c.JSON(status, body)
}

// decodeJSON читает тело запроса в v, при ошибке сам отвечает 400.
//
// Ошибку биндинга нельзя отдавать в DomainErrorToResponse: она не доменная, не
// матчится ни на один case и уходила в default — то есть клиент получал 500 за
// свой же кривой JSON, причём молча, без записи в лог. Наружу отдаём общий
// текст, подробности только в лог.
func (h *Handlers) decodeJSON(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(v); err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{"method": c.Request.Method, "path": c.Request.URL.Path}).
			Debug("handlers: invalid request body")
		c.JSON(http.StatusBadRequest, &dto.ErrorResponse{Code: "bad_request", Message: "invalid request body"})
		return false
	}
	return true
}

// parseIDParam читает числовой параметр пути, при ошибке сам пишет 400.
func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(errors.DomainErrorToResponse(err))
		return 0, false
	}
	return id, true
}

func parsePagination(c *gin.Context) (offset, limit int) {
	offset, _ = strconv.Atoi(c.Query("offset"))
	if offset < 0 {
		offset = 0
	}
	limit, _ = strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	return offset, limit
}

// parseOptionalIDQuery — необязательный числовой query-параметр, пусто/битое значение = nil.
func parseOptionalIDQuery(c *gin.Context, name string) *int64 {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

func parseOptionalStatusQuery(c *gin.Context, name string) *models.PurchaseStatus {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	status := models.PurchaseStatus(raw)
	return &status
}

func parseOptionalMerchantQuery(c *gin.Context, name string) *models.Merchant {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	merchant := models.Merchant(raw)
	return &merchant
}

// parseOptionalTimeQuery — необязательная дата в формате RFC3339.
func parseOptionalTimeQuery(c *gin.Context, name string) *time.Time {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}
