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
// @Summary Create a new payment
// @Description Submit a new payment transaction. Requires an idempotency key to prevent double charging.
// @Security BearerAuth
// @Tags Payments
// @Accept json
// @Produce json
// @Param Idempotency-Key header string true "Idempotency key"
// @Param request body createPaymentRequest true "Payment details"
// @Success 201 {object} domain.Payment "Created payment details"
// @Failure 400 {object} map[string]string "Bad request details"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Idempotency key conflict"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /payments [post]
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
// @Summary Get payment by ID
// @Description Query specific payment details by UUID. Uses Redis cache lookup before DB query.
// @Security BearerAuth
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path string true "Payment UUID"
// @Success 200 {object} domain.Payment "Payment details"
// @Failure 400 {object} map[string]string "Invalid UUID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Payment not found error"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /payments/{id} [get]
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
// @Summary List all payments
// @Description Fetch all payment transactions ordered by creation date descending.
// @Security BearerAuth
// @Tags Payments
// @Accept json
// @Produce json
// @Success 200 {array} domain.Payment "List of payments"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /payments [get]
func (h *PaymentHandler) List(c *gin.Context) {
	payments, err := h.paymentService.ListPayments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query payments list"})
		return
	}

	c.JSON(http.StatusOK, payments)
}
