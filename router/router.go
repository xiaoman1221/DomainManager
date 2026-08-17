package router

import (
	"DomainManager/handlers"
	"DomainManager/middleware"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		// Public site-wide configuration (footer / SNS links) for the login page.
		api.GET("/site/config", handlers.GetSiteConfig)

		auth := api.Group("/auth")
		{
			auth.POST("/register", middleware.RateLimitAuth(), handlers.Register)
			auth.POST("/login", middleware.RateLimitAuth(), handlers.Login)
			auth.GET("/profile", middleware.AuthRequired(), handlers.GetProfile)

			// OauthGo third-party login (https://o.1v.fit/docs)
			auth.GET("/oauth/providers", handlers.OauthGoProviders)
			auth.POST("/oauth/login", middleware.RateLimitAuth(), handlers.OauthGoLogin)
			auth.POST("/oauth/ticket", middleware.RateLimitAuth(), handlers.OauthGoRedeem)
			auth.GET("/oauth/callback", handlers.OauthGoCallback)
			// Third-party account binding (profile page)
			auth.GET("/oauth/bindings", middleware.AuthRequired(), handlers.OauthGoBindings)
			auth.POST("/oauth/bind", middleware.AuthRequired(), handlers.OauthGoBindStart)
			auth.GET("/oauth/bind/callback/:token", handlers.OauthGoBindCallback)
			auth.DELETE("/oauth/bind/:provider", middleware.AuthRequired(), handlers.OauthGoUnbind)
		}

		domains := api.Group("/domains", middleware.AuthRequired())
		{
			domains.GET("", handlers.ListDomains)
			domains.GET("/stats", handlers.GetDomainStats)
			domains.POST("", handlers.CreateDomain)
			domains.GET("/export", handlers.ExportDomains)
			domains.POST("/import", handlers.ImportDomainsCSV)
			domains.GET("/:id", handlers.GetDomain)
			domains.PUT("/:id", handlers.UpdateDomain)
			domains.DELETE("/:id", handlers.DeleteDomain)
			domains.POST("/batch-delete", handlers.BatchDeleteDomains)
			domains.POST("/batch-update", handlers.BatchUpdateDomains)
			domains.POST("/:id/refresh", handlers.RefreshDomainInfo)
			domains.GET("/:id/renewal-price", handlers.QueryRenewalPrice)
			domains.POST("/batch-renewal-price", handlers.BatchQueryRenewalPrice)
		}

		whois := api.Group("/whois", middleware.AuthRequired())
		{
			whois.GET("", handlers.QueryWhois)
			whois.GET("/digitalplat-suffixes", handlers.GetDigitalPlatSuffixes)
		}

		icp := api.Group("/icp", middleware.AuthRequired())
		{
			icp.GET("", handlers.QueryICP)
		}

		price := api.Group("/price", middleware.AuthRequired())
		{
			price.POST("/compare", handlers.ComparePrices)
			price.GET("/tlds", handlers.GetSupportedTLDs)
			price.POST("/refresh", handlers.RefreshPrices)
		}

		registrars := api.Group("/registrars", middleware.AuthRequired())
		{
			registrars.GET("", handlers.ListRegistrars)
			registrars.GET("/types", handlers.GetRegistrarTypes)
			registrars.GET("/:id", handlers.GetRegistrar)
			registrars.POST("", handlers.CreateRegistrar)
			registrars.PUT("/:id", handlers.UpdateRegistrar)
			registrars.DELETE("/:id", handlers.DeleteRegistrar)
			registrars.POST("/import", handlers.ImportDomainsFromRegistrar)
			registrars.GET("/export", handlers.ExportRegistrars)
			registrars.POST("/import-csv", handlers.ImportRegistrarsCSV)
		}

		certificates := api.Group("/certificates", middleware.AuthRequired())
		{
			certificates.GET("", handlers.ListCertificates)
			certificates.GET("/stats", handlers.GetCertificateStats)
			certificates.POST("", handlers.CreateCertificate)
			certificates.GET("/certimate/config", middleware.AdminRequired(), handlers.GetCertimateConfig)
			certificates.POST("/certimate/config", middleware.AdminRequired(), handlers.SaveCertimateConfig)
			certificates.POST("/certimate/sync", middleware.AdminRequired(), handlers.SyncCertimateCertificates)
			certificates.GET("/:id", handlers.GetCertificate)
			certificates.PUT("/:id", handlers.UpdateCertificate)
			certificates.DELETE("/:id", handlers.DeleteCertificate)
		}

		notifications := api.Group("/notifications", middleware.AuthRequired())
		{
			notifications.GET("/types", handlers.GetNotificationTypes)
			notifications.GET("/channels", handlers.ListNotificationChannels)
			notifications.POST("/channels", handlers.CreateNotificationChannel)
			notifications.PUT("/channels/:id", handlers.UpdateNotificationChannel)
			notifications.DELETE("/channels/:id", handlers.DeleteNotificationChannel)
			notifications.POST("/channels/:id/toggle", handlers.ToggleNotificationChannel)
			notifications.POST("/channels/:id/test", handlers.TestNotificationChannel)
			notifications.GET("/logs", handlers.ListNotificationLogs)
			notifications.POST("/send-expiry", handlers.SendDomainExpiryNotifications)
		}

		settings := api.Group("/settings", middleware.AuthRequired())
		{
			// System-wide settings and user management are admin-only.
			settings.GET("", middleware.AdminRequired(), handlers.GetSystemSettings)
			settings.PUT("", middleware.AdminRequired(), handlers.UpdateSystemSettings)
			settings.GET("/info", middleware.AdminRequired(), handlers.GetSystemInfo)
			settings.PUT("/profile", handlers.UpdateUserProfile)
			settings.PUT("/password", handlers.ChangePassword)
			settings.GET("/users", middleware.AdminRequired(), handlers.ListAllUsers)
			settings.PUT("/users/:id", middleware.AdminRequired(), handlers.UpdateUser)
			settings.DELETE("/users/:id", middleware.AdminRequired(), handlers.DeleteUser)
			settings.PUT("/users/:id/role-group", middleware.AdminRequired(), handlers.UpdateUserRoleGroup)
			settings.PUT("/users/:id/user-group", middleware.AdminRequired(), handlers.UpdateUserGroup)
			settings.PUT("/users/:id/password", middleware.AdminRequired(), handlers.AdminUpdateUserPassword)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
