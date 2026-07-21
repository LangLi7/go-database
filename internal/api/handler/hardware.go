package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/hardware"
	"go-database/internal/recipe"
)

// HandleHardwareScan returns the host hardware spec (RAM/CPU/GPU).
func HandleHardwareScan() gin.HandlerFunc {
	return func(c *gin.Context) {
		spec := hardware.Scan(c.Request.Context())
		response.Success(c, spec)
	}
}

// HandleRecipeList lists available recipes.
func HandleRecipeList() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, recipe.List())
	}
}

// HandleRecipeRun executes a recipe by name with JSON body as input.
func HandleRecipeRun() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		var in map[string]any
		if err := c.ShouldBindJSON(&in); err != nil {
			// allow empty input
			in = map[string]any{}
		}
		out, err := recipe.Run(name, in)
		if err != nil {
			response.Error(c, http.StatusNotFound, "recipe_not_found", err.Error())
			return
		}
		response.Success(c, out)
	}
}
