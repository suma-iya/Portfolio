package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/suma-iya/portfolio-backend/internal/logger"
	"github.com/suma-iya/portfolio-backend/internal/models"
	"github.com/suma-iya/portfolio-backend/internal/services"
)

type ContactHandler struct {
	mailer *services.Mailer
	logger *logger.Logger
}

func NewContactHandler(mailer *services.Mailer, log *logger.Logger) *ContactHandler {
	return &ContactHandler{mailer: mailer, logger: log}
}

func (h *ContactHandler) Submit(c *gin.Context) {
	var req models.ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// The frontend shows this message to visitors, so keep validator internals out of it.
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Please check your name, email and message (at least 10 characters) and try again.",
		})
		return
	}

	if err := h.mailer.SendContactMessage(req); err != nil {
		h.logger.Error("failed to send contact email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Could not send your message right now. Please try again later or email directly.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully."})
}
