package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indrawanalghifary/whatsgo/internal/config"
	"github.com/indrawanalghifary/whatsgo/internal/session"
	"github.com/skip2/go-qrcode"
)

type APIHandler struct {
	sessionManager *session.Manager
	config         *config.Config
}

type CreateSessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

type SendMessageRequest struct {
	JID     string `json:"jid" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewAPIHandler(sessionManager *session.Manager, config *config.Config) *APIHandler {
	return &APIHandler{
		sessionManager: sessionManager,
		config:         config,
	}
}

func (h *APIHandler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	session, err := h.sessionManager.CreateSession(req.SessionID)
	if err != nil {
		c.JSON(http.StatusConflict, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Connect session
	if err := session.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to connect session: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "Session created successfully",
		Data: gin.H{
			"session_id": req.SessionID,
			"status":     session.GetStatus(),
		},
	})
}

func (h *APIHandler) ListSessions(c *gin.Context) {
	sessions := h.sessionManager.ListSessions()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    sessions,
	})
}

func (h *APIHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"session_id": sessionID,
			"status":     session.GetStatus(),
			"logged_in":  session.IsLoggedIn(),
		},
	})
}

func (h *APIHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	if err := h.sessionManager.DeleteSession(sessionID); err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Session deleted successfully",
	})
}

func (h *APIHandler) GetQRCode(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// If already logged in, return error
	if session.IsLoggedIn() {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Session is already logged in",
		})
		return
	}

	// Wait for QR code with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.config.QRCode.Timeout)*time.Second)
	defer cancel()

	select {
	case qrCode := <-session.QRChan:
		// Generate QR code image
		_, err := h.generateQRImage(sessionID, qrCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "Failed to generate QR code image: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Data: gin.H{
				"qr_code": qrCode,
				"qr_url":  fmt.Sprintf("/qr/%s.png", sessionID),
				"expires": time.Now().Add(time.Duration(h.config.QRCode.Timeout) * time.Second),
			},
		})
		return
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, APIResponse{
			Success: false,
			Error:   "QR code generation timeout",
		})
		return
	}
}

func (h *APIHandler) LogoutSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Logout and disconnect
	if session.Client.IsConnected() {
		if err := session.Client.Logout(context.Background()); err != nil {
			c.JSON(http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "Failed to logout: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Session logged out successfully",
	})
}

func (h *APIHandler) SendMessage(c *gin.Context) {
	sessionID := c.Param("sessionId")

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := session.SendMessage(req.JID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to send message: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Message sent successfully",
	})
}

func (h *APIHandler) GetChats(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if !session.Client.IsConnected() {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Session is not connected",
		})
		return
	}

	// For now, return a placeholder response
	// In a real implementation, you would need to implement chat history storage
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"message": "Chat history feature not implemented yet",
			"chats":   []interface{}{},
		},
	})
}

func (h *APIHandler) GetSessionStatus(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"session_id": sessionID,
			"status":     session.GetStatus(),
			"logged_in":  session.IsLoggedIn(),
			"connected":  session.Client.IsConnected(),
		},
	})
}

func (h *APIHandler) generateQRImage(sessionID, qrCode string) (string, error) {
	// Create qr_codes directory if it doesn't exist
	qrDir := "./qr_codes"
	if err := os.MkdirAll(qrDir, 0755); err != nil {
		return "", err
	}

	// Generate QR code image
	qrPath := filepath.Join(qrDir, sessionID+".png")
	err := qrcode.WriteFile(qrCode, qrcode.Medium, 256, qrPath)
	if err != nil {
		return "", err
	}

	return qrPath, nil
}