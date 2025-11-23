package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/iweapp/errors"
	"github.com/techagentng/iweapp/server/response"
)

// handleAIAnalyze handles AI analysis of optional document text with a user prompt
// Request JSON: { "prompt": string, "documentText": string }
// Response JSON: { "answer": string }
func (s *Server) handleAIAnalyze() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Prompt       string `json:"prompt"`
			DocumentText string `json:"documentText"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.JSON(c, "Invalid request data", http.StatusBadRequest, nil, errors.New(err.Error(), http.StatusBadRequest))
			return
		}

		if strings.TrimSpace(req.Prompt) == "" {
			response.JSON(c, "'prompt' is required", http.StatusBadRequest, nil, errors.New("prompt is required", http.StatusBadRequest))
			return
		}

		if s.AIService == nil {
			response.JSON(c, "AI service not initialized", http.StatusServiceUnavailable, nil, errors.New("ai service not configured", http.StatusServiceUnavailable))
			return
		}

		answer, err := s.AIService.AnalyzeDocument(c.Request.Context(), req.DocumentText, req.Prompt)
		if err != nil {
			response.JSON(c, "AI analysis failed", http.StatusBadGateway, nil, errors.New(err.Error(), http.StatusBadGateway))
			return
		}

		response.JSON(c, "OK", http.StatusOK, gin.H{"answer": answer}, nil)
	}
}
