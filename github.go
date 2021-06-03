package main

import (
	"fmt"
	"os"

	b64 "encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/iver-wharf/wharf-api-client-go/pkg/wharfapi"
	_ "github.com/iver-wharf/wharf-provider-github/docs"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

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
// @Failure 400 {object} string "Bad request"
// @Failure 401 {object} string "Unauthorized or missing jwt token"
// @Router /github [post]
func runGitHubHandler(c *gin.Context) {
	i := importBody{}
	err := c.BindJSON(&i)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	fmt.Println("from json: ", i)

	ctx := context.Background()
	importer := githubImporter{
		Context: ctx,
		WharfClient: wharfapi.Client{
			ApiUrl:     os.Getenv("WHARF_API_URL"),
			AuthHeader: c.GetHeader("Authorization"),
		},
	}

	importer.Provider, err = importer.getProvider(i)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to get provider. %+v", err))
		return
	}

	importer.Token, err = importer.getToken(i)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to get token. %+v", err))
		return
	}

	importer.GithubClient, err = importer.initGithubConnection()
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to init github connection. %+v", err))
		return
	}

	if i.ProjectID != 0 || i.Project != "" {
		err = importer.importProject(i)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to import project. %+v", err))
			return
		}
	} else {
		err = importer.importGroup(i.Group)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to import organization or group. %+v", err))
			return
		}
	}

	c.Status(http.StatusCreated)
}

func (importer githubImporter) getProvider(i importBody) (wharfapi.Provider, error) {
	var provider wharfapi.Provider
	var err error
	if i.ProviderID != 0 {
		provider, err = importer.WharfClient.GetProviderById(i.ProviderID)
		if err != nil {
			return provider, err
		} else if provider.ProviderID == 0 {
			err = fmt.Errorf("provider with id %v not found", i.ProviderID)
		} else if provider.URL != i.URL {
			err = fmt.Errorf("invalid url in provider %v", provider.URL)
		} else if provider.UploadURL != i.UploadURL {
			err = fmt.Errorf("invalid upload url in provider %v", provider.UploadURL)
		}
	} else {
		provider, err = importer.WharfClient.GetProvider("github", i.URL, i.UploadURL, importer.Token.TokenID)
		if err != nil || provider.ProviderID == 0 {
			provider, err = importer.WharfClient.PostProvider(wharfapi.Provider{Name: "github", URL: i.URL, UploadURL: i.UploadURL})
		}
	}
	fmt.Println("Provider from db: ", provider)
	return provider, nil
}

func (importer githubImporter) getToken(i importBody) (wharfapi.Token, error) {
	var token wharfapi.Token
	var err error

	if importer.Provider.ProviderID == 0 {
		return token, fmt.Errorf("provider not found")
	}

	if i.TokenID != 0 {
		token, err = importer.WharfClient.GetTokenById(i.TokenID)
		if err != nil {
			return token, err
		} else if token.TokenID == 0 {
			err = fmt.Errorf(fmt.Sprintf("Token with id %v not found", i.TokenID))
		} else if token.ProviderID != importer.Provider.ProviderID {
			err = fmt.Errorf(fmt.Sprintf("Token with invalid provider id %v.", i.ProviderID))
		}
	} else {
		token, err = importer.WharfClient.GetToken(i.Token, i.User)
		if err != nil || token.TokenID == 0 {
			token, err = importer.WharfClient.PostToken(wharfapi.Token{Token: i.Token, UserName: i.User, ProviderID: importer.Provider.ProviderID})
		}
	}

	fmt.Println("Token from db: ", token)
	return token, err
}

func (importer githubImporter) initGithubConnection() (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: importer.Token.Token})
	tc := oauth2.NewClient(importer.Context, ts)
	client, err := github.NewEnterpriseClient(importer.Provider.URL, importer.Provider.UploadURL, tc)
	return client, err
}

func (importer githubImporter) getBuildDefiniton(owner string, projectName string) string {
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

func (importer githubImporter) importProject(i importBody) error {
	if i.ProjectID != 0 {
		project, err := importer.WharfClient.GetProjectById(i.ProjectID)
		if err != nil {
			return err
		} else if project.ProjectID == 0 {
			return fmt.Errorf(fmt.Sprintf("Project with id %v not found.", i.ProjectID))
		}
		i.Project = project.Name
	}

	var repo *github.Repository
	var err error
	if i.Group != "" {
		repo, _, err = importer.GithubClient.Repositories.Get(importer.Context, i.Group, i.Project)
		if err != nil {
			return err
		} else if repo.GetName() != i.Project {
			return fmt.Errorf(fmt.Sprintf("Project with name %v not found.", i.Project))
		} else if repo.GetOwner().GetLogin() != i.Group {
			return fmt.Errorf(fmt.Sprintf("Unable to find project with name %v in organization or associeted with user %v.",
				i.Project, repo.GetOwner().GetLogin()))
		}
	} else {
		repos, _, err := importer.GithubClient.Repositories.List(importer.Context, "", nil)
		if err != nil {
			return err
		}

		for _, repository := range repos {
			if repository.GetName() == i.Project {
				repo = repository
			}
		}
	}

	return importer.putProject(repo)
}

func (importer githubImporter) putProject(repo *github.Repository) error {
	buildDefinitionStr := importer.getBuildDefiniton(repo.GetOwner().GetLogin(), repo.GetName())

	project, err := importer.WharfClient.PutProject(
		wharfapi.Project{
			Name:            repo.GetName(),
			TokenID:         importer.Token.TokenID,
			GroupName:       repo.GetOwner().GetLogin(),
			BuildDefinition: buildDefinitionStr,
			Description:     repo.GetDescription(),
			ProviderID:      importer.Provider.ProviderID})
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
			err = importer.putProject(repo)
			if err != nil {
				return err
			}
		}
	}

	return err
}
