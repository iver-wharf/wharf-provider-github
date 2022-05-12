package main

import (
	"errors"
	"fmt"
	"strconv"

	b64 "encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/iver-wharf/wharf-api-client-go/v2/pkg/model/request"
	"github.com/iver-wharf/wharf-api-client-go/v2/pkg/model/response"
	"github.com/iver-wharf/wharf-api-client-go/v2/pkg/wharfapi"

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
	Provider     response.Provider
	Token        response.Token
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

	importer.Provider, err = importer.getProvider(i)
	if err != nil {
		ginutil.WriteAPIClientReadError(c, err,
			fmt.Sprintf("Unable to get GitHub provider by ID %v or name %q", i.ProviderID, i.Provider))
		return
	}

	importer.GithubClient, err = importer.initGithubConnection()
	if err != nil {
		ginutil.WriteAPIClientReadError(c, err,
			fmt.Sprintf("Unable to parse provider url %q",
				importer.Provider.URL))
		return
	}

	if i.ProjectID != 0 {
		err = importer.refreshProject(i)
		if err != nil {
			ginutil.WriteAPIClientWriteError(c, err,
				fmt.Sprintf("Unable to refresh project %q with ID %d from GitHub.", i.Project, i.ProjectID))
			return
		}
		// If a refresh is possible the project already exists. Don't create a new project from group/name.
		c.Status(http.StatusOK)
		return
	}

	if i.Project != "" {
		err = importer.importProject(i)
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

func (importer githubImporter) getTokenWritesProblem(c *gin.Context, i importBody) (response.Token, bool) {
	var token response.Token
	var err error

	if i.TokenID != 0 {
		token, err = importer.WharfClient.GetToken(i.TokenID)
		if err != nil {
			ginutil.WriteAPIClientReadError(c, err,
				fmt.Sprintf(
					"Unable to get token by ID %d. Likely because of a failed request or malformed response.",
					i.TokenID))
			return response.Token{}, false
		} else if token.TokenID == 0 {
			err = fmt.Errorf("token with ID %d not found", i.TokenID)
			ginutil.WriteAPIClientReadError(c, err,
				fmt.Sprintf("Token with ID %d not found.", i.TokenID))
		}
	} else {
		token, err = importer.WharfClient.CreateToken(request.Token{Token: i.Token, UserName: i.User})
		if err != nil {
			ginutil.WriteAPIClientWriteError(c, err,
				fmt.Sprintf(
					"Unable to create token for user %q. Likely because of a failed request or malformed response.",
					i.User))
			return response.Token{}, false
		}
	}

	log.Debug().WithUint("tokenId", token.TokenID).Message("Found token from DB.")
	return token, true
}

func (importer githubImporter) getProvider(i importBody) (response.Provider, error) {
	var provider response.Provider
	var err error

	if i.ProviderID != 0 {
		provider, err = importer.WharfClient.GetProvider(i.ProviderID)
		if err != nil {
			return provider, err
		} else if provider.ProviderID == 0 {
			err = fmt.Errorf("provider with id %v not found", i.ProviderID)
		} else if provider.URL != i.URL {
			err = fmt.Errorf("invalid url in provider %q", provider.URL)
		}
	} else {
		provider, err = importer.WharfClient.CreateProvider(request.Provider{Name: "github", URL: i.URL, TokenID: importer.Token.TokenID})
	}
	log.Debug().
		WithUint("providerId", provider.ProviderID).
		WithString("providerName", string(provider.Name)).
		Message("Found provider from DB.")
	return provider, err
}

func (importer githubImporter) initGithubConnection() (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: importer.Token.Token})
	tc := oauth2.NewClient(importer.Context, ts)
	client, err := github.NewEnterpriseClient(importer.Provider.URL, "", tc)
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

func (importer githubImporter) refreshProject(i importBody) error {
	if i.ProjectID == 0 {
		return errors.New("can't refresh project without project ID")
	}

	project, err := importer.WharfClient.GetProject(i.ProjectID)
	if err != nil {
		return err
	} else if project.ProjectID == 0 {
		return fmt.Errorf("project with id %d not found", i.ProjectID)
	}
	i.Project = project.Name

	repo, err := importer.getRepo(i.Group, i.Project)
	if err != nil {
		return err
	}

	buildDefinitionStr := importer.getBuildDefinition(repo.GetOwner().GetLogin(), repo.GetName())
	projectUpdate := request.ProjectUpdate{
		Name:            repo.GetName(),
		TokenID:         importer.Token.TokenID,
		GroupName:       repo.GetOwner().GetLogin(),
		BuildDefinition: buildDefinitionStr,
		Description:     repo.GetDescription(),
		AvatarURL:       *repo.GetOwner().AvatarURL,
		ProviderID:      importer.Provider.ProviderID,
		GitURL:          *repo.SSHURL}
	_, err = importer.WharfClient.UpdateProject(i.ProjectID, projectUpdate)
	return err
}

func (importer githubImporter) importProject(i importBody) error {
	if i.ProjectID != 0 {
		return fmt.Errorf("import project failed: ID should be 0, was %d", i.ProjectID)
	}

	var repo *github.Repository
	var err error
	if i.Group != "" {
		repo, err = importer.getRepo(i.Group, i.Project)
		if err != nil {
			return err
		}
	} else {
		repos, _, err := importer.GithubClient.Repositories.List(importer.Context, "", nil)
		if err != nil {
			return err
		}

		for _, repository := range repos {
			if repository.GetName() == i.Project {
				repo = repository
				break
			}
		}
	}

	return importer.createProject(repo)
}

func (importer githubImporter) getRepo(group, project string) (*github.Repository, error) {
	repo, _, err := importer.GithubClient.Repositories.Get(importer.Context, group, project)
	if err != nil {
		return nil, err
	} else if repo.GetName() != project {
		return nil, fmt.Errorf("project with name %q not found", project)
	} else if repo.GetOwner().GetLogin() != group {
		return nil, fmt.Errorf("unable to find project with name %q in organization or associated with user %q",
			project, repo.GetOwner().GetLogin())
	}
	return repo, nil
}

func (importer githubImporter) createProject(repo *github.Repository) error {
	buildDefinitionStr := importer.getBuildDefinition(repo.GetOwner().GetLogin(), repo.GetName())
	project := request.Project{
		Name:            repo.GetName(),
		TokenID:         importer.Token.TokenID,
		GroupName:       repo.GetOwner().GetLogin(),
		BuildDefinition: buildDefinitionStr,
		Description:     repo.GetDescription(),
		AvatarURL:       *repo.GetOwner().AvatarURL,
		ProviderID:      importer.Provider.ProviderID,
		GitURL:          *repo.SSHURL,
		RemoteProjectID: strconv.FormatInt(repo.GetID(), 10),
	}

	newProject, err := importer.WharfClient.CreateProject(project)
	if err != nil {
		return err
	} else if newProject.ProjectID == 0 {
		return fmt.Errorf("unable to create project '%s/%s'", project.GroupName, project.Name)
	}

	branches, _, err := importer.GithubClient.Repositories.ListBranches(importer.Context, newProject.GroupName, newProject.Name, nil)
	if err != nil {
		return err
	}
	for _, branch := range branches {
		_, err := importer.WharfClient.CreateProjectBranch(
			newProject.ProjectID,
			request.Branch{
				Name:    branch.GetName(),
				Default: branch.GetName() == repo.GetDefaultBranch(),
			})
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
			err = importer.createProject(repo)
			if err != nil {
				return err
			}
		}
	}

	return err
}
