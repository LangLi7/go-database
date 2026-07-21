package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
	"go-database/internal/samples"
)

// ListSamples returns all available sample database templates
func ListSamples() gin.HandlerFunc {
	return func(c *gin.Context) {
		names, err := samples.List()
		if err != nil {
			response.InternalError(c, "failed to list samples: "+err.Error())
			return
		}

		type sampleInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		var infos []sampleInfo
		for _, name := range names {
			s, err := samples.Get(name)
			if err != nil {
				continue
			}
			infos = append(infos, sampleInfo{Name: name, Description: s.Description})
		}

		response.Success(c, infos)
	}
}

// LoadSample loads a sample database into a connection
func LoadSample(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		sampleName := c.Param("sample")

		s, err := samples.Get(sampleName)
		if err != nil {
			response.NotFound(c, "sample not found: "+sampleName)
			return
		}

		if err := s.Load(c.Request.Context(), mgr, connID); err != nil {
			response.Error(c, http.StatusBadGateway, "SAMPLE_FAILED", err.Error())
			return
		}

		response.Created(c, gin.H{
			"sample": sampleName,
			"name":   s.Name,
			"tables": len(s.Tables),
		})
	}
}

// ImportData imports tables and data from a JSON definition
func ImportData(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")

		var s samples.Sample
		if err := c.ShouldBindJSON(&s); err != nil {
			response.BadRequest(c, "invalid import JSON: "+err.Error())
			return
		}

		if len(s.Tables) == 0 {
			response.BadRequest(c, "import must contain at least one table")
			return
		}

		if err := s.Load(c.Request.Context(), mgr, connID); err != nil {
			response.Error(c, http.StatusBadGateway, "IMPORT_FAILED", err.Error())
			return
		}

		response.Created(c, gin.H{
			"tables": len(s.Tables),
		})
	}
}
