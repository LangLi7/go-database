package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/llm"
)

// HandleLocalModels returns available GGUF models from LM Studio.
func HandleLocalModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		models, err := llm.FetchLMStudioModels(ctx, "http://localhost:1234")
		if err != nil {
			slog.Debug("lm studio not available", "error", err)
			response.Success(c, []llm.LocalModel{})
			return
		}
		response.Success(c, models)
	}
}

// HandleRemoteModels returns available (free) models from OpenRouter.
func HandleRemoteModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("Authorization")
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		models, err := llm.FetchOpenRouterModels(ctx, apiKey)
		if err != nil {
			slog.Debug("openrouter models not available", "error", err)
			// fallback: return hardcoded free list
			var fallback []llm.RemoteModel
			for _, id := range llm.FreeModels {
				fallback = append(fallback, llm.RemoteModel{ID: id, Pricing: llm.ModelPricing{Prompt: "0", Completion: "0"}})
			}
			response.Success(c, fallback)
			return
		}

		var free []llm.RemoteModel
		for _, m := range models {
			if m.IsFree() {
				free = append(free, m)
			}
		}
		response.Success(c, free)
	}
}
