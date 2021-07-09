package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/logger"
	"github.com/iver-wharf/wharf-core/pkg/logger/consolepretty"
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
	URL       string `json:"url" example:"https://api.github.com/"`
	UploadURL string `json:"uploadUrl" example:""`
	// used in refresh only
	ProviderID uint `json:"providerId" example:"0"`
	// azuredevops, gitlab or github
	Provider string `json:"provider" example:"github"`
	// used in refresh only
	ProjectID uint   `json:"projectId" example:"0"`
	Project   string `json:"project" example:"wharf-provider-github"`
	Group     string `json:"group" example:"iver-wharf"`
}

const buildDefinitionFileName = ".wharf-ci.yml"

var log = logger.NewScoped("WHARF-PROVIDER-GITHUB")

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
	logger.AddOutput(logger.LevelDebug, consolepretty.Default)

	if err := loadEmbeddedVersionFile(); err != nil {
		log.Error().WithError(err).Message("Failed to read embedded version.yaml.")
		os.Exit(1)
	}

	docs.SwaggerInfo.Version = AppVersion.Version

	gin.DefaultWriter = ginutil.DefaultLoggerWriter
	gin.DefaultErrorWriter = ginutil.DefaultLoggerWriter

	r := gin.New()
	r.Use(
		ginutil.DefaultLoggerHandler,
		gin.Recovery(),
	)

	allowCors, ok := os.LookupEnv("ALLOW_CORS")
	if ok && allowCors == "YES" {
		log.Info().Message("Allowing all origins in CORS.")
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
