package main

import (
	"fmt"
	"strconv"

	b64 "encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/iver-wharf/wharf-api-client-go/pkg/wharfapi"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	_ "github.com/iver-wharf/wharf-provider-github/docs"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type githubImporterModule struct {
	config *Config
}

func (m githubImporterModule) register(r gin.IRoutes) {
	r.POST("/import/github", m.runGitHubHandler)
}

type githubImporter struct {
	GithubClient *github.Client
	WharfClient  wharfapi.Client
	Context      context.Context
	Provider     wharfapi.Provider
	Token        wharfapi.Token
}

// runGitHubHandler godoc
// @Summary Import projects from GitHub or refresh existing one
// @Accept json
// @Produce json
// @Param import body importBody _ "import object"
// @Success 201 "Successfully imported"
// @Failure 400 {object} problem.Response "Bad request"
// @Failure 401 {object} problem.Response "Unauthorized or missing jwt token"
// @Failure 502 {object} problem.Response "Bad gateway"
// @Router /github [post]
func (m githubImporterModule) runGitHubHandler(c *gin.Context) {
	i := importBody{}
	err := c.ShouldBindJSON(&i)
	if err != nil {
		ginutil.WriteInvalidBindError(c, err,
			"One or more parameters failed to parse when reading the request body for GitHub projects import/refresh")
		return
	}

	ctx := context.Background()
	importer := githubImporter{
		Context: ctx,
		WharfClient: wharfapi.Client{
			APIURL:     m.config.API.URL,
			AuthHeader: c.GetHeader("Authorization"),
		},
	}

	var ok bool
	importer.Token, ok = importer.getTokenWritesProblem(c, i)
	if !ok {
		return
	}

	importer.Provider, err = importer.getProvider(i, importer.Token)
	if err != nil {
		ginutil.WriteAPIClientReadError(c, err,
			fmt.Sprintf("Unable to get GitHub provider by ID %v or name %q", i.ProviderID, i.Provider))
		return
	}

	importer.GithubClient, err = importer.initGithubConnection()
	if err != nil {
		ginutil.WriteAPIClientReadError(c, err,
			fmt.Sprintf("Unable to parse provider url %q or upload url %q",
				importer.Provider.URL, importer.Provider.UploadURL))
		return
	}

	if i.ProjectID != 0 {
		err = importer.refreshProject(i.ProjectID)
		if err != nil {
			ginutil.WriteAPIClientWriteError(c, err,
				fmt.Sprintf("Unable to refresh project %q with ID %d from GitHub.", i.Project, i.ProjectID))
			return
		}
	} else if i.Project != "" {
		err = importer.importProject(i.Group, i.Project)
		if err != nil {
			ginutil.WriteAPIClientWriteError(c, err,
				fmt.Sprintf("Unable to import project %q with ID %d from GitHub.", i.Project, i.ProjectID))
			return
		}
	} else {
		err = importer.importGroup(i.Group)
		if err != nil {
			ginutil.WriteAPIClientWriteError(c, err,
				fmt.Sprintf("Unable to import group %q from GitHub.", i.Group))
			return
		}
	}

	c.Status(http.StatusCreated)
}

func (importer githubImporter) getTokenWritesProblem(c *gin.Context, i importBody) (wharfapi.Token, bool) {
	var token wharfapi.Token
	var err error

	if i.TokenID != 0 {
		token, err = importer.WharfClient.GetTokenByID(i.TokenID)
		if err != nil {
			ginutil.WriteAPIClientReadError(c, err,
				fmt.Sprintf(
					"Unable to get token by ID %d. Likely because of a failed request or malformed response.",
					i.TokenID))
			return wharfapi.Token{}, false
		} else if token.TokenID == 0 {
			err = fmt.Errorf("token with ID %d not found", i.TokenID)
			ginutil.WriteAPIClientReadError(c, err,
				fmt.Sprintf("Token with ID %d not found.", i.TokenID))
		}
	} else {
		token, err = importer.WharfClient.PutToken(wharfapi.Token{Token: i.Token, UserName: i.User})
		if err != nil {
			ginutil.WriteAPIClientWriteError(c, err,
				fmt.Sprintf(
					"Unable to create token for user %q. Likely because of a failed request or malformed response.",
					i.User))
			return wharfapi.Token{}, false
		}
	}

	log.Debug().WithUint("tokenId", token.TokenID).Message("Found token from DB.")
	return token, true
}

func (importer githubImporter) getProvider(i importBody, token wharfapi.Token) (wharfapi.Provider, error) {
	var provider wharfapi.Provider
	var err error

	if i.ProviderID != 0 {
		provider, err = importer.WharfClient.GetProviderByID(i.ProviderID)
		if err != nil {
			return provider, err
		} else if provider.ProviderID == 0 {
			err = fmt.Errorf("provider with id %v not found", i.ProviderID)
		} else if provider.URL != i.URL {
			err = fmt.Errorf("invalid url in provider %q", provider.URL)
		} else if provider.UploadURL != i.UploadURL {
			err = fmt.Errorf("invalid upload url in provider %q", provider.UploadURL)
		}
	} else {
		provider, err = importer.WharfClient.PutProvider(wharfapi.Provider{Name: "github", URL: i.URL, UploadURL: i.UploadURL, TokenID: token.TokenID})
	}
	log.Debug().
		WithUint("providerId", provider.ProviderID).
		WithString("providerName", provider.Name).
		Message("Found provider from DB.")
	return provider, err
}

func (importer githubImporter) initGithubConnection() (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: importer.Token.Token})
	tc := oauth2.NewClient(importer.Context, ts)
	client, err := github.NewEnterpriseClient(importer.Provider.URL, importer.Provider.UploadURL, tc)
	return client, err
}

func (importer githubImporter) getBuildDefinition(owner string, projectName string) string {
	fileContent, _, _, err := importer.GithubClient.Repositories.GetContents(importer.Context, owner, projectName, buildDefinitionFileName, nil)
	if err != nil {
		return ""
	}

	bodyString, err := b64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		return ""
	}

	return string(bodyString)
}

func (importer githubImporter) refreshProject(projectID uint) error {
	project, err := importer.WharfClient.GetProjectByID(projectID)
	if err != nil {
		return err
	} else if project.ProjectID == 0 {
		return fmt.Errorf("project with id %v not found", projectID)
	}
	remoteProjectID, err := strconv.ParseInt(project.RemoteProjectID, 0, strconv.IntSize)
	if err != nil {
		return err
	}
	return importer.importProjectByRemoteProjectID(project.ProjectID, remoteProjectID)
}

func (importer githubImporter) importProjectByRemoteProjectID(projectID uint, remoteProjectID int64) error {
	repo, _, err := importer.GithubClient.Repositories.GetByID(importer.Context, remoteProjectID)
	if err != nil {
		return err
	}

	return importer.putProject(projectID, repo)
}

func (importer githubImporter) importProject(groupName, projectName string) error {
	var repo *github.Repository
	var err error
	if groupName != "" {
		repo, _, err = importer.GithubClient.Repositories.Get(importer.Context, groupName, projectName)
		if err != nil {
			return err
		} else if repo.GetName() != projectName {
			return fmt.Errorf("project with name %q not found", projectName)
		} else if repo.GetOwner().GetLogin() != groupName {
			return fmt.Errorf("unable to find project with name %q in organization or associated with user %q",
				projectName, repo.GetOwner().GetLogin())
		}
	} else {
		repos, _, err := importer.GithubClient.Repositories.List(importer.Context, "", nil)
		if err != nil {
			return err
		}

		for _, repository := range repos {
			if repository.GetName() == projectName {
				repo = repository
				break
			}
		}
	}

	return importer.putProject(0, repo)
}

func (importer githubImporter) putProject(projectID uint, repo *github.Repository) error {
	log.Debug().
		WithString("repoName", repo.GetName()).
		WithString("groupName", repo.GetOrganization().GetName()).
		WithInt64("remoteProjectID", repo.GetID()).
		WithUint("projectID", projectID).
		Message("Called putProject")
	buildDefinitionStr := importer.getBuildDefinition(repo.GetOwner().GetLogin(), repo.GetName())
	project, err := importer.WharfClient.PutProject(
		wharfapi.Project{
			Name:            repo.GetName(),
			TokenID:         importer.Token.TokenID,
			GroupName:       repo.GetOwner().GetLogin(),
			BuildDefinition: buildDefinitionStr,
			Description:     repo.GetDescription(),
			AvatarURL:       *repo.GetOwner().AvatarURL,
			ProviderID:      importer.Provider.ProviderID,
			GitURL:          *repo.GitURL,
			ProjectID:       projectID,
			RemoteProjectID: fmt.Sprintf("%d", repo.GetID()),
		})
	if err != nil {
		return err
	} else if project.ProjectID == 0 {
		return fmt.Errorf("unable to put project")
	}

	branches, _, err := importer.GithubClient.Repositories.ListBranches(importer.Context, project.GroupName, project.Name, nil)
	if err != nil {
		return err
	}
	for _, branch := range branches {
		_, err := importer.WharfClient.PutBranch(
			wharfapi.Branch{
				Name:      branch.GetName(),
				ProjectID: project.ProjectID,
				Default:   branch.GetName() == repo.GetDefaultBranch(),
				TokenID:   importer.Token.TokenID})
		if err != nil {
			break
		}
	}

	return err
}

func (importer githubImporter) importGroup(groupName string) error {
	repos, _, err := importer.GithubClient.Repositories.List(importer.Context, groupName, nil)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if groupName == "" || repo.GetOwner().GetLogin() == groupName {
			err = importer.putProject(0, repo)
			if err != nil {
				return err
			}
		}
	}

	return err
}
