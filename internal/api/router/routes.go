package router

import (
	"github.com/gin-gonic/gin"

	"go-database/internal/api/handler"
	"go-database/internal/api/middleware"
	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/internaldb"
)

// permMW wraps PermissionMiddleware with a role loader that queries the store
func permMW(store *internaldb.Store, perm string) gin.HandlerFunc {
	return middleware.PermissionMiddleware(perm, func(c *gin.Context, name string) *auth.Role {
		role, err := store.GetRole(c.Request.Context(), name)
		if err != nil {
			return nil
		}
		return role
	})
}

func SetupRoutes(r *gin.Engine, store *internaldb.Store, connMgr *connection.Manager, jwt *auth.JWTService, apikeySvc *auth.APIKeyService) {

	r.POST("/api/v1/auth/login", middleware.LoginRateLimit(), handler.Login(store, jwt))
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok", "version": "0.1.0"})
	})

	authGroup := r.Group("/api/v1")
	authGroup.Use(middleware.AuthMiddleware(middleware.AuthConfig{JWT: jwt, APIKey: apikeySvc}))
	{
		authGroup.POST("/auth/refresh", handler.RefreshToken(jwt))
		authGroup.GET("/auth/verify", handler.VerifyToken(jwt))
		authGroup.POST("/auth/change-password", handler.ChangePassword(store))

		connGroup := authGroup.Group("")
		connGroup.Use(permMW(store, auth.PermConnectionsList))
		{
			connGroup.GET("/connections", handler.ListConnections(connMgr))
			connGroup.GET("/connections/:id", handler.GetConnection(connMgr))
			connGroup.GET("/connections/:id/ping", handler.PingConnection(connMgr))
			connGroup.GET("/connections/:id/tables", handler.ListTables(connMgr))
			connGroup.GET("/connections/:id/schema", handler.GetSchema(connMgr))
			connGroup.POST("/connections/:id/query", handler.QueryConnection(connMgr))
			connGroup.POST("/connections/:id/execute", handler.ExecuteConnection(connMgr))
			connGroup.GET("/connections/:id/databases", handler.ListDatabases(connMgr))
		}
		authGroup.POST("/databases/standalone", permMW(store, auth.PermConnectionsCreate), handler.CreateStandaloneDatabase(connMgr))
		connGroup.POST("/connections", permMW(store, auth.PermConnectionsCreate), handler.CreateConnection(connMgr))
		connGroup.DELETE("/connections/:id", permMW(store, auth.PermConnectionsDelete), handler.DeleteConnection(connMgr))
		connGroup.POST("/connections/:id/databases", permMW(store, auth.PermConnectionsExec), handler.CreateDatabase(connMgr))
		connGroup.DELETE("/connections/:id/databases/:name", permMW(store, auth.PermConnectionsExec), handler.DropDatabase(connMgr))
		connGroup.POST("/connections/:id/tables", permMW(store, auth.PermConnectionsExec), handler.CreateTable(connMgr))
		connGroup.DELETE("/connections/:id/tables/:name", permMW(store, auth.PermConnectionsExec), handler.DropTable(connMgr))

		explorerGroup := authGroup.Group("/connections/:id")
		explorerGroup.Use(permMW(store, auth.PermConnectionsQuery))
		{
			explorerGroup.GET("/browse/:table", handler.BrowseTable(connMgr))
			explorerGroup.POST("/row/:table", handler.InsertRow(connMgr))
			explorerGroup.PUT("/row/:table/:pk/:val", handler.UpdateRow(connMgr))
			explorerGroup.DELETE("/row/:table/:pk/:val", handler.DeleteRow(connMgr))
		}

		adminGroup := authGroup.Group("/admin")
		{
			adminGroup.GET("/stats", permMW(store, auth.PermTrafficView), handler.GetStats(connMgr))
			adminGroup.GET("/design", permMW(store, auth.PermSettingsRead), handler.GetDesign(store))
			adminGroup.POST("/design", permMW(store, auth.PermSettingsWrite), handler.SaveDesign(store))
			adminGroup.GET("/activity", permMW(store, auth.PermTrafficView), handler.GetActivity(store))
		}
		userGroup := authGroup.Group("/admin")
		userGroup.Use(permMW(store, auth.PermUsersList))
		{
			userGroup.GET("/users", handler.ListUsers(store))
			userGroup.POST("/users", permMW(store, auth.PermUsersCreate), handler.CreateUser(store))
			userGroup.PUT("/users/:id", permMW(store, auth.PermUsersEdit), handler.UpdateUser(store))
			userGroup.DELETE("/users/:id", permMW(store, auth.PermUsersDelete), handler.DeleteUser(store))
			userGroup.GET("/users/:id/permissions", handler.GetUserPermissions(store))
			userGroup.PUT("/users/:id/permissions", permMW(store, auth.PermUsersEdit), handler.SetUserPermissions(store))
		}
		roleGroup := authGroup.Group("/admin/roles")
		roleGroup.Use(permMW(store, auth.PermRolesManage))
		{
			roleGroup.GET("", handler.ListRoles(store))
			roleGroup.POST("", handler.CreateRole(store))
			roleGroup.PUT("/:id", handler.UpdateRole(store))
			roleGroup.DELETE("/:id", handler.DeleteRole(store))
			roleGroup.PUT("/:id/permissions", handler.SetRolePermissions(store))
		}

		permGroup := authGroup.Group("/admin")
		permGroup.Use(permMW(store, auth.PermRolesManage))
		{
			permGroup.GET("/permission-groups", handler.GetPermissionGroups())
			permGroup.GET("/users/:id/db-access", handler.GetUserDBAccess(store))
			permGroup.PUT("/users/:id/db-access", handler.SetUserDBAccess(store))
		}

		trafficGroup := authGroup.Group("/traffic")
		trafficGroup.Use(permMW(store, auth.PermTrafficView))
		{
			trafficGroup.GET("/stats", handler.GetStats(connMgr))
			trafficGroup.GET("/requests", handler.GetRequests(connMgr))
		}

		keyGroup := authGroup.Group("/apikeys")
		keyGroup.Use(permMW(store, auth.PermAPIKeysManage))
		{
			keyGroup.GET("", handler.ListAPIKeys(store))
			keyGroup.POST("", handler.CreateAPIKey(store))
			keyGroup.DELETE("/:prefix", handler.DeleteAPIKey(store))
		}

		transferGroup := authGroup.Group("/transfer")
		transferGroup.Use(permMW(store, auth.PermConnectionsExec))
		{
			transferGroup.POST("", handler.StartTransfer())
			transferGroup.GET("/:id", handler.GetTransferStatus())
			transferGroup.DELETE("/:id", handler.CancelTransfer())
			transferGroup.GET("/:id/log", handler.GetTransferLog())
		}

		suggestGroup := authGroup.Group("/suggest")
		suggestGroup.Use(permMW(store, auth.PermConnectionsList))
		{
			suggestGroup.POST("", handler.GetSuggestions(connMgr))
		}

		execGroup := authGroup.Group("/execute")
		execGroup.Use(permMW(store, auth.PermConnectionsQuery))
		{
			execGroup.POST("/safe", handler.ExecuteSafe(connMgr))
		}

		// WebSocket — streaming queries
		wsQueryGroup := authGroup.Group("/ws/query")
		wsQueryGroup.Use(permMW(store, auth.PermConnectionsQuery))
		{
			wsQueryGroup.GET("/:id", handler.WSQueryHandler(connMgr))
		}

		// SSE — real-time event streams
		sseGroup := authGroup.Group("/sse")
		{
			sseGroup.GET("/activity", permMW(store, auth.PermTrafficView), handler.SSEActivityHandler(connMgr))
			sseGroup.GET("/stats", permMW(store, auth.PermTrafficView), handler.SSEStatsHandler(connMgr))
		}

		// Samples — database templates
		samplesGroup := authGroup.Group("/samples")
		{
			samplesGroup.GET("", handler.ListSamples())
		}

		// Sample loading + Import per connection
		loadGroup := explorerGroup.Group("")
		loadGroup.Use(permMW(store, auth.PermConnectionsExec))
		{
			loadGroup.POST("/samples/:sample", handler.LoadSample(connMgr))
			loadGroup.POST("/import", handler.ImportData(connMgr))
		}
	}
}
