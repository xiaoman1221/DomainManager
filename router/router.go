package router

import (
	"DomainManager/handlers"
	"DomainManager/middleware"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()

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
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.GET("/profile", middleware.AuthRequired(), handlers.GetProfile)
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
			certificates.GET("/certimate/config", handlers.GetCertimateConfig)
			certificates.POST("/certimate/config", handlers.SaveCertimateConfig)
			certificates.POST("/certimate/sync", handlers.SyncCertimateCertificates)
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
			settings.GET("", handlers.GetSystemSettings)
			settings.PUT("", handlers.UpdateSystemSetting)
			settings.GET("/info", handlers.GetSystemInfo)
			settings.PUT("/profile", handlers.UpdateUserProfile)
			settings.PUT("/password", handlers.ChangePassword)
			settings.GET("/users", handlers.ListAllUsers)
			settings.PUT("/users/:id/role", handlers.UpdateUserRole)
			settings.PUT("/users/:id/password", handlers.AdminUpdateUserPassword)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
