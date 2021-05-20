package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-provider-github/docs"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type importBody struct {
	// used in refresh only
	TokenID   uint   `json:"tokenId" example:"0"`
	Token     string `json:"token" example:"sample token"`
	User      string `json:"user" example:"sample user name"`
	URL       string `json:"url" example:"https://gitlab.local"`
	UploadURL string `json:"uploadUrl" example:""`
	// used in refresh only
	ProviderID uint `json:"providerId" example:"0"`
	// azuredevops, gitlab or github
	Provider string `json:"provider" example:"github"`
	// used in refresh only
	ProjectID uint   `json:"projectId" example:"0"`
	Project   string `json:"project" example:"sample project name (wharf-api)"`
	Group     string `json:"group" example:"default (iver-wharf)"`
}

const buildDefinitionFileName = ".wharf-ci.yml"

// @title Wharf provider API for GitHub
// @description Wharf backend API for integrating GitHub repositories with
// @description the Wharf main API.
// @license.name MIT
// @license.url https://github.com/iver-wharf/wharf-provider-github/blob/master/LICENSE
// @contact.name Iver Wharf GitHub provider API support
// @contact.url https://github.com/iver-wharf/wharf-provider-github/issues
// @contact.email wharf@iver.se
// @basePath /import
func main() {
	if err := loadEmbeddedVersionFile(); err != nil {
		fmt.Println("Failed to read embedded version.yaml file:", err)
		os.Exit(1)
	}

	docs.SwaggerInfo.Version = AppVersion.Version

	r := gin.Default()

	allowCors, ok := os.LookupEnv("ALLOW_CORS")
	if ok && allowCors == "YES" {
		fmt.Printf("Allowing CORS\n")
		r.Use(cors.Default())
	}

	r.GET("/", runPingHandler)
	r.POST("/import/github", runGitHubHandler)
	r.GET("/import/github/version", getVersionHandler)
	r.GET("/import/github/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	_ = r.Run(getBindAddress())
}

func getBindAddress() string {
	bindAddress, isBindAddressDefined := os.LookupEnv("BIND_ADDRESS")
	if !isBindAddressDefined || bindAddress == "" {
		return "0.0.0.0:8080"
	}
	return bindAddress
}

func runPingHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}
