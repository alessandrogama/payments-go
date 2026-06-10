package http

import (
	"errors"
	"net/http"

	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService application.AuthService
}

// NewAuthHandler creates a new instance of AuthHandler.
func NewAuthHandler(authService application.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register handles user registration request.
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body registerRequest true "Registration credentials"
// @Success 201 {object} map[string]interface{} "User ID and Email"
// @Failure 400 {object} map[string]string "Bad request details"
// @Failure 409 {object} map[string]string "User already exists error"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": user.Email,
	})
}

// Login validates user credentials and returns a JWT token.
// @Summary Login user
// @Description Authenticate a user with email and password and return a JWT access token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body loginRequest true "Login credentials"
// @Success 200 {object} map[string]string "Access token"
// @Failure 400 {object} map[string]string "Bad request details"
// @Failure 401 {object} map[string]string "Invalid credentials error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}
