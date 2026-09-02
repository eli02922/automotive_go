package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/embuscado/automotive-catalog/internal/api/handlers"
	"github.com/embuscado/automotive-catalog/internal/api/middleware"
)

func Register(r *gin.Engine, catalog *handlers.CatalogHandler, search *handlers.SearchHandler, jwtSecret string) {
	r.GET("/health", search.HealthCheck)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.BearerAuth(jwtSecret))
	{
		// Product catalog
		products := v1.Group("/products")
		{
			products.POST("", catalog.CreateProduct)
			products.GET("", catalog.ListProducts)
			products.GET("/:id", catalog.GetProduct)
			products.PUT("/:id", catalog.UpdateProduct)
			products.DELETE("/:id", middleware.RequireRole("admin"), catalog.DeleteProduct)

			// Fitment (ACES application data)
			products.GET("/:id/fitments", catalog.GetFitments)
			products.POST("/:id/fitments", catalog.UpsertFitment)
		}

		// Vehicle-first fitment search
		search_ := v1.Group("/search")
		{
			search_.GET("/vehicle", search.VehicleSearch)
		}
	}
}
