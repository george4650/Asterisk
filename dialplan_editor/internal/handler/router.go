package handler

import (
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "myapp/docs"

	"myapp/internal/usecase"
)

func NewRouter(router *gin.Engine, os usecase.AsteriskUseCases) {

	asteriskHandlers := &AsteriskHandler{
		us: os,
	}

	// Routers
	router.GET("/get-files", asteriskHandlers.GetFiles)

	router.GET("/create-dir", asteriskHandlers.CreateDir)

	router.POST("/create-dir", asteriskHandlers.CreateDir)

	router.POST("/upload-files", asteriskHandlers.UploadFiles)

	router.GET("/get-audio", asteriskHandlers.GetAudio)

	router.GET("/get-script", asteriskHandlers.GetScript)

	router.POST("/update-script", asteriskHandlers.UpdateScript)


	router.GET("/get-root", asteriskHandlers.GetRoot) //возможность локально скопировать корневую папку (в задании нет)

	// Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
