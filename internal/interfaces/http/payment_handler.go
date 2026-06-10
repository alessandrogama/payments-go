package http

import (
	"errors"
	"net/http"

	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentService application.PaymentService
}

// NewPaymentHandler creates a new instance of PaymentHandler.
func NewPaymentHandler(paymentService application.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

type createPaymentRequest struct {
	CustomerID string  `json:"customer_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	Currency   string  `json:"currency" binding:"required"`
}

// Create submits a new payment processing request.
func (h *PaymentHandler) Create(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
		return
	}

	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	custID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id, expected UUID"})
		return
	}

	input := application.CreatePaymentInput{
		CustomerID:     custID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		IdempotencyKey: idempotencyKey,
	}

	payment, err := h.paymentService.CreatePayment(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAmount) || errors.Is(err, domain.ErrInvalidCurrency) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrIdempotencyFailed) {
			c.JSON(http.StatusConflict, gin.H{"error": "idempotency conflict"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process payment request"})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

// GetByID retrieves a payment by its unique ID.
func (h *PaymentHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID, expected UUID"})
		return
	}

	payment, err := h.paymentService.GetPaymentByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payment"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// List queries and lists all payment records in descending creation order.
func (h *PaymentHandler) List(c *gin.Context) {
	payments, err := h.paymentService.ListPayments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query payments list"})
		return
	}

	c.JSON(http.StatusOK, payments)
}
