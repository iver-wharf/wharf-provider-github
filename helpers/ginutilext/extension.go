package ginutilext

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/problem"
)

func WriteAPIReadError(c *gin.Context, err error, detail string) {
	ginutil.WriteProblemError(c, err, problem.Response{
		Type: "/prob/provider/unexpected-api-read-error",
		Title: "Unexpected API read error.",
		Status: http.StatusBadGateway,
		Detail: detail,
	})
}

func WriteAPIWriteError(c *gin.Context, err error, detail string) {
	ginutil.WriteProblemError(c, err, problem.Response{
		Type: "/prob/provider/unexpected-api-write-error",
		Title: "Unexpected API write error.",
		Status: http.StatusBadGateway,
		Detail: detail,
	})
}

func WriteResponseFormatError(c *gin.Context, err error, detail string) {
	ginutil.WriteProblemError(c, err, problem.Response{
		Type: "/prob/provider/unexpected-response-format",
		Title: "Unexpected response format.",
		Status: http.StatusBadGateway,
		Detail: detail,
	})
}

func WriteFetchBuildDefinitionError(c *gin.Context, err error, detail string) {
	ginutil.WriteProblemError(c, err, problem.Response{
		Type: "/prob/provider/fetch-build-definition",
		Title: "Error fetching build definition.",
		Status: http.StatusBadRequest,
		Detail: detail,
	})
}

func WriteComposingProviderDataError(c *gin.Context, err error, detail string) {
	ginutil.WriteProblemError(c, err, problem.Response{
		Type: "/prob/provider/composing-provider-data",
		Title: "Error composing provider data.",
		Status: http.StatusBadRequest,
		Detail: detail,
	})
}

func WriteTriggerError(c *gin.Context, err error, detail string) {
	ginutil.WriteProblemError(c, err, problem.Response{
		Type: "/prob/provider/unexpected-trigger-error",
		Title: "Unexpected trigger error.",
		Status: http.StatusBadGateway,
		Detail: detail,
	})
}