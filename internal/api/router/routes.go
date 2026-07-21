package router

import (
	"context"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/handler"
	"go-database/internal/api/middleware"
	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/crypto"
	"go-database/internal/internaldb"
	"go-database/internal/llm"
	"go-database/internal/scheduler"
	"go-database/internal/transfer"
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

// dbAccessMW wraps DBAccessMiddleware with a role loader
func dbAccessMW(store *internaldb.Store) gin.HandlerFunc {
	return middleware.DBAccessMiddleware(func(c *gin.Context, name string) *auth.Role {
		role, err := store.GetRole(c.Request.Context(), name)
		if err != nil {
			return nil
		}
		return role
	})
}

func SetupRoutes(r *gin.Engine, store *internaldb.Store, connMgr *connection.Manager, jwt *auth.JWTService, apikeySvc *auth.APIKeyService, transferEngine transfer.TransferEngine, sched *scheduler.Scheduler, schedStore scheduler.SchedulerStore, cryptoSvc *crypto.Service) {

	r.POST("/api/v1/auth/login", middleware.LoginRateLimit(), handler.Login(store, jwt))
	// Passkey (WebAuthn) public login ceremony
	r.POST("/api/v1/auth/passkeys/login/begin", handler.PasskeyLoginBegin(store))
	r.POST("/api/v1/auth/passkeys/login/finish", handler.PasskeyLoginFinish(store, jwt))
	r.GET("/api/v1/setup/status", handler.SetupStatus(store))
	r.POST("/api/v1/setup/initialize", handler.InitializeSetup(store))
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok", "version": "0.1.0"})
	})

	authGroup := r.Group("/api/v1")
	authGroup.Use(middleware.AuthMiddleware(middleware.AuthConfig{JWT: jwt, APIKey: apikeySvc}))
	{
		authGroup.POST("/auth/refresh", handler.RefreshToken(jwt))
		authGroup.GET("/auth/verify", handler.VerifyToken(jwt))
		authGroup.POST("/auth/change-password", handler.ChangePassword(store))

		// Passkeys (WebAuthn) — register/list/delete require auth; login is public
		authGroup.GET("/auth/passkeys", handler.PasskeyList(store))
		authGroup.POST("/auth/passkeys/register/begin", handler.PasskeyRegisterBegin(store))
		authGroup.POST("/auth/passkeys/register/finish", handler.PasskeyRegisterFinish(store))
		authGroup.DELETE("/auth/passkeys/:id", handler.PasskeyDelete(store))

		connGroup := authGroup.Group("")
		connGroup.Use(permMW(store, auth.PermConnectionsList))
		{
			connGroup.GET("/connections", handler.ListConnections(connMgr))
		}

		// Connection-specific routes with DBAccess check
		connIDGroup := authGroup.Group("")
		connIDGroup.Use(dbAccessMW(store))
		{
			connIDGroup.GET("/connections/:id", permMW(store, auth.PermConnectionsList), handler.GetConnection(connMgr))
			connIDGroup.GET("/connections/:id/ping", permMW(store, auth.PermConnectionsList), handler.PingConnection(connMgr))
			connIDGroup.GET("/connections/:id/tables", permMW(store, auth.PermConnectionsList), handler.ListTables(connMgr))
			connIDGroup.GET("/connections/:id/schema", permMW(store, auth.PermConnectionsList), handler.GetSchema(connMgr))
			connIDGroup.GET("/connections/:id/databases", permMW(store, auth.PermConnectionsList), handler.ListDatabases(connMgr))
			connIDGroup.POST("/connections/:id/query", permMW(store, auth.PermConnectionsQuery), handler.QueryConnection(connMgr))
			connIDGroup.POST("/connections/:id/execute", permMW(store, auth.PermConnectionsExec), handler.ExecuteConnection(connMgr))
			connIDGroup.POST("/connections/:id/databases", permMW(store, auth.PermConnectionsExec), handler.CreateDatabase(connMgr))
			connIDGroup.DELETE("/connections/:id/databases/:name", permMW(store, auth.PermConnectionsExec), handler.DropDatabase(connMgr))
			connIDGroup.POST("/connections/:id/tables", permMW(store, auth.PermConnectionsExec), handler.CreateTable(connMgr))
			connIDGroup.DELETE("/connections/:id/tables/:name", permMW(store, auth.PermConnectionsExec), handler.DropTable(connMgr))
		}
		authGroup.POST("/databases/standalone", permMW(store, auth.PermConnectionsCreate), handler.CreateStandaloneDatabase(connMgr))
		connGroup.POST("/connections", permMW(store, auth.PermConnectionsCreate), handler.CreateConnection(connMgr))
		connGroup.POST("/connections/test", permMW(store, auth.PermConnectionsCreate), handler.TestConnection(connMgr))
		connGroup.DELETE("/connections/:id", permMW(store, auth.PermConnectionsDelete), handler.DeleteConnection(connMgr))

		explorerGroup := authGroup.Group("/connections/:id")
		explorerGroup.Use(dbAccessMW(store))
		{
			explorerGroup.GET("/browse/:table", permMW(store, auth.PermConnectionsQuery), handler.BrowseTable(connMgr))
			explorerGroup.POST("/row/:table", permMW(store, auth.PermConnectionsExec), handler.InsertRow(connMgr))
			explorerGroup.PUT("/row/:table/:pk/:val", permMW(store, auth.PermConnectionsExec), handler.UpdateRow(connMgr))
			explorerGroup.DELETE("/row/:table/:pk/:val", permMW(store, auth.PermConnectionsExec), handler.DeleteRow(connMgr))
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
			transferGroup.POST("", handler.StartTransfer(connMgr, transferEngine))
			transferGroup.GET("/:id", handler.GetTransferStatus(transferEngine))
			transferGroup.DELETE("/:id", handler.CancelTransfer(transferEngine))
			transferGroup.GET("/:id/log", handler.GetTransferLog(transferEngine))
		}

		suggestGroup := authGroup.Group("/suggest")
		suggestGroup.Use(permMW(store, auth.PermConnectionsList))
		{
			suggestGroup.POST("", handler.GetSuggestions(connMgr))
			suggestGroup.POST("/ai", handler.AISuggest(connMgr))
		}

		execGroup := authGroup.Group("/execute")
		execGroup.Use(permMW(store, auth.PermConnectionsQuery))
		{
			execGroup.POST("/safe", handler.ExecuteSafe(connMgr))
		}

		// ── Per-type endpoints (alias for direct DB access, B) ──
		// POST /api/v1/db/{type}/query  — throwaway SELECT
		// POST /api/v1/db/{type}/execute — throwaway write
		// POST /api/v1/db/{type}/test    — connectivity check
		dbTypeGroup := authGroup.Group("/db")
		{
			dbTypeGroup.POST("/:type/query", permMW(store, auth.PermConnectionsQuery), handler.DBTypeQuery(connMgr))
			dbTypeGroup.POST("/:type/execute", permMW(store, auth.PermConnectionsExec), handler.DBTypeExecute(connMgr))
			dbTypeGroup.POST("/:type/test", permMW(store, auth.PermConnectionsCreate), handler.DBTypeTest(connMgr))
		}

		// WebSocket — streaming queries
		wsQueryGroup := authGroup.Group("/ws/query")
		wsQueryGroup.Use(permMW(store, auth.PermConnectionsQuery))
		{
			wsQueryGroup.GET("/:id", handler.WSQueryHandler(connMgr))
		}

		// WebSocket — transfer progress
		wsTransferGroup := authGroup.Group("/ws/transfer")
		wsTransferGroup.Use(permMW(store, auth.PermConnectionsExec))
		{
			wsTransferGroup.GET("/:id", handler.WSTransferProgress(transferEngine))
		}

		// SSE — real-time event streams
		sseGroup := authGroup.Group("/sse")
		{
			sseGroup.GET("/activity", permMW(store, auth.PermTrafficView), handler.SSEActivityHandler(connMgr))
			sseGroup.GET("/stats", permMW(store, auth.PermTrafficView), handler.SSEStatsHandler(connMgr))
		}

		// Scheduled jobs
		scheduleGroup := authGroup.Group("/schedules")
		scheduleGroup.Use(permMW(store, auth.PermConnectionsExec))
		{
			scheduleGroup.GET("", handler.ListSchedules(sched, schedStore))
			scheduleGroup.GET("/:id", handler.GetSchedule(sched, schedStore))
			scheduleGroup.POST("", handler.CreateSchedule(sched, schedStore))
			scheduleGroup.PUT("/:id", handler.UpdateSchedule(sched, schedStore))
			scheduleGroup.DELETE("/:id", handler.DeleteSchedule(sched, schedStore))
		}

		// Encryption
		cryptoGroup := authGroup.Group("/crypto")
		cryptoGroup.Use(permMW(store, auth.PermConnectionsExec))
		{
			cryptoGroup.GET("/keys", handler.ListCryptoKeys(cryptoSvc))
			cryptoGroup.POST("/keys", handler.CreateCryptoKey(cryptoSvc))
			cryptoGroup.DELETE("/keys/:id", handler.DeleteCryptoKey(cryptoSvc))
			cryptoGroup.POST("/encrypt", handler.EncryptData(cryptoSvc))
			cryptoGroup.POST("/decrypt", handler.DecryptData(cryptoSvc))
			// New: signing, hashing, discovery, rotation
			cryptoGroup.POST("/sign", handler.SignData(cryptoSvc))
			cryptoGroup.POST("/verify", handler.VerifyData(cryptoSvc))
			cryptoGroup.POST("/hash", handler.HashData(cryptoSvc))
			cryptoGroup.POST("/keys/rotate", handler.RotateCryptoKey(cryptoSvc))
			cryptoGroup.GET("/algorithms", handler.ListCryptoAlgorithms(cryptoSvc))
		}

		// Column-level encryption on a connection
		cryptoColGroup := explorerGroup.Group("/crypto")
		cryptoColGroup.Use(permMW(store, auth.PermConnectionsExec))
		{
			cryptoColGroup.POST("/encrypt/:table/:column", handler.EncryptColumn(cryptoSvc, connMgr))
			cryptoColGroup.POST("/decrypt/:table/:column", handler.DecryptColumn(cryptoSvc, connMgr))
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
			loadGroup.POST("/import/csv", handler.ImportCSV(connMgr))
		}
	}

	// Public model discovery (no auth required)
	r.GET("/api/v1/models/local", handler.HandleLocalModels())
	r.GET("/api/v1/models/remote", handler.HandleRemoteModels())

	// AI Agent chat + SSE
	logFn := func(action, details string) {
		_ = store.LogAudit(context.Background(), "system", action, details)
	}
	r.POST("/api/v1/agent/chat", handler.HandleAgentChat(logFn))
	r.GET("/api/v1/agent/stream", handler.HandleAgentStream(logFn))

	// AI Setup wizard
	r.POST("/api/v1/setup/ai", handler.HandleAISetup())

	// SQL Templates
	r.GET("/api/v1/templates", handler.ListTemplates())
	r.POST("/api/v1/templates/apply", handler.ApplyTemplate(connMgr))

	// Model download + start (llama.cpp)
	r.POST("/api/v1/models/download", handler.DownloadModel())
	r.POST("/api/v1/models/start", handler.StartModel(llm.FindLlamaCPP))

	// Hardware compatibility cookbook
	r.GET("/api/v1/hardware", handler.HandleHardwareScan())
	r.POST("/api/v1/hardware/submit", handler.HandleHardwareSubmit())
	r.GET("/api/v1/recipes", handler.HandleRecipeList())
	r.POST("/api/v1/recipes/:name", handler.HandleRecipeRun())

	// Documentation server (renders docs/*.md as HTML)
	r.GET("/docs", handler.HandleDocsRedirect)
	r.GET("/docs/*slug", handler.HandleDocs)
}
