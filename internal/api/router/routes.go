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

func SetupRoutes(r *gin.Engine, store *internaldb.Store, connMgr *connection.Manager, jwt *auth.JWTService) {

	r.POST("/api/v1/auth/login", handler.Login(store, jwt))
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok", "version": "0.1.0"})
	})

	authGroup := r.Group("/api/v1")
	authGroup.Use(middleware.AuthMiddleware(jwt))
	{
		authGroup.POST("/auth/refresh", handler.RefreshToken(jwt))
		authGroup.POST("/auth/change-password", handler.ChangePassword(store))

		connGroup := authGroup.Group("")
		connGroup.Use(middleware.PermissionMiddleware(auth.PermConnectionsList))
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
		authGroup.POST("/databases/standalone", middleware.PermissionMiddleware(auth.PermConnectionsCreate), handler.CreateStandaloneDatabase(connMgr))
		connGroup.POST("/connections", middleware.PermissionMiddleware(auth.PermConnectionsCreate), handler.CreateConnection(connMgr))
		connGroup.DELETE("/connections/:id", middleware.PermissionMiddleware(auth.PermConnectionsDelete), handler.DeleteConnection(connMgr))
		connGroup.POST("/connections/:id/databases", middleware.PermissionMiddleware(auth.PermConnectionsExec), handler.CreateDatabase(connMgr))
		connGroup.DELETE("/connections/:id/databases/:name", middleware.PermissionMiddleware(auth.PermConnectionsExec), handler.DropDatabase(connMgr))
		connGroup.POST("/connections/:id/tables", middleware.PermissionMiddleware(auth.PermConnectionsExec), handler.CreateTable(connMgr))
		connGroup.DELETE("/connections/:id/tables/:name", middleware.PermissionMiddleware(auth.PermConnectionsExec), handler.DropTable(connMgr))

		explorerGroup := authGroup.Group("/connections/:id")
		explorerGroup.Use(middleware.PermissionMiddleware(auth.PermConnectionsQuery))
		{
			explorerGroup.GET("/browse/:table", handler.BrowseTable(connMgr))
			explorerGroup.POST("/row/:table", handler.InsertRow(connMgr))
			explorerGroup.PUT("/row/:table/:pk/:val", handler.UpdateRow(connMgr))
			explorerGroup.DELETE("/row/:table/:pk/:val", handler.DeleteRow(connMgr))
		}

		adminGroup := authGroup.Group("/admin")
		{
			adminGroup.GET("/stats", middleware.PermissionMiddleware(auth.PermTrafficView), handler.GetStats(connMgr))
			adminGroup.GET("/design", middleware.PermissionMiddleware(auth.PermSettingsRead), handler.GetDesign(store))
			adminGroup.POST("/design", middleware.PermissionMiddleware(auth.PermSettingsWrite), handler.SaveDesign(store))
			adminGroup.GET("/activity", middleware.PermissionMiddleware(auth.PermTrafficView), handler.GetActivity(store))
		}
		userGroup := authGroup.Group("/admin")
		userGroup.Use(middleware.PermissionMiddleware(auth.PermUsersList))
		{
			userGroup.GET("/users", handler.ListUsers(store))
			userGroup.POST("/users", middleware.PermissionMiddleware(auth.PermUsersCreate), handler.CreateUser(store))
			userGroup.PUT("/users/:id", middleware.PermissionMiddleware(auth.PermUsersEdit), handler.UpdateUser(store))
			userGroup.DELETE("/users/:id", middleware.PermissionMiddleware(auth.PermUsersDelete), handler.DeleteUser(store))
			userGroup.GET("/users/:id/permissions", handler.GetUserPermissions(store))
			userGroup.PUT("/users/:id/permissions", middleware.PermissionMiddleware(auth.PermUsersEdit), handler.SetUserPermissions(store))
		}
		roleGroup := authGroup.Group("/admin/roles")
		roleGroup.Use(middleware.PermissionMiddleware(auth.PermRolesManage))
		{
			roleGroup.GET("", handler.ListRoles(store))
			roleGroup.POST("", handler.CreateRole(store))
			roleGroup.PUT("/:id", handler.UpdateRole(store))
			roleGroup.DELETE("/:id", handler.DeleteRole(store))
			roleGroup.PUT("/:id/permissions", handler.SetRolePermissions(store))
		}

		permGroup := authGroup.Group("/admin")
		permGroup.Use(middleware.PermissionMiddleware(auth.PermRolesManage))
		{
			permGroup.GET("/permission-groups", handler.GetPermissionGroups())
			permGroup.GET("/users/:id/db-access", handler.GetUserDBAccess(store))
			permGroup.PUT("/users/:id/db-access", handler.SetUserDBAccess(store))
		}

		trafficGroup := authGroup.Group("/traffic")
		trafficGroup.Use(middleware.PermissionMiddleware(auth.PermTrafficView))
		{
			trafficGroup.GET("/stats", handler.GetStats(connMgr))
			trafficGroup.GET("/requests", handler.GetRequests(connMgr))
		}

		keyGroup := authGroup.Group("/apikeys")
		keyGroup.Use(middleware.PermissionMiddleware(auth.PermAPIKeysManage))
		{
			keyGroup.GET("", handler.ListAPIKeys(store))
			keyGroup.POST("", handler.CreateAPIKey(store))
			keyGroup.DELETE("/:prefix", handler.DeleteAPIKey(store))
		}

		transferGroup := authGroup.Group("/transfer")
		transferGroup.Use(middleware.PermissionMiddleware(auth.PermConnectionsExec))
		{
			transferGroup.POST("", handler.StartTransfer())
			transferGroup.GET("/:id", handler.GetTransferStatus())
			transferGroup.DELETE("/:id", handler.CancelTransfer())
			transferGroup.GET("/:id/log", handler.GetTransferLog())
		}

		suggestGroup := authGroup.Group("/suggest")
		suggestGroup.Use(middleware.PermissionMiddleware(auth.PermConnectionsList))
		{
			suggestGroup.POST("", handler.GetSuggestions(connMgr))
		}

		execGroup := authGroup.Group("/execute")
		execGroup.Use(middleware.PermissionMiddleware(auth.PermConnectionsQuery))
		{
			execGroup.POST("/safe", handler.ExecuteSafe(connMgr))
		}
	}
}
