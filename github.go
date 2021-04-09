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

type GithubImporter struct {
	GithubClient *github.Client
	WharfClient  wharfapi.Client
	Context      context.Context
	Provider     wharfapi.Provider
	Token        wharfapi.Token
}

// RunGithubHandler godoc
// @Summary Import projects from github or refresh existing one
// @Accept  json
// @Produce  json
// @Param import body main.Import _ "import object"
// @Success 200 "OK"
// @Failure 400 "Bad request"
// @Failure 401 "Unauthorized or missing jwt token"
// @Router /github [post]
func RunGithubHandler(c *gin.Context) {
	i := Import{}
	err := c.BindJSON(&i)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	fmt.Println("from json: ", i)

	ctx := context.Background()
	importer := GithubImporter{
		Context: ctx,
		WharfClient: wharfapi.Client{
			ApiUrl:     os.Getenv("WHARF_API_URL"),
			AuthHeader: c.GetHeader("Authorization"),
		},
	}

	importer.Provider, err = importer.GetProvider(i)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to get provider. %+v", err))
		return
	}

	importer.Token, err = importer.GetToken(i)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to get token. %+v", err))
		return
	}

	importer.GithubClient, err = importer.InitGithubConnection()
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to init github connection. %+v", err))
		return
	}

	if i.ProjectId != 0 || i.Project != "" {
		err = importer.ImportProject(i)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to import project. %+v", err))
			return
		}
	} else {
		err = importer.Import(i.Group)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to import organization or group. %+v", err))
			return
		}
	}

	c.JSON(http.StatusOK, "OK")
}

func (base GithubImporter) GetProvider(i Import) (wharfapi.Provider, error) {
	var provider wharfapi.Provider
	var err error
	if i.ProviderId != 0 {
		provider, err = base.WharfClient.GetProviderById(i.ProviderId)
		if err != nil {
			return provider, err
		} else if provider.ProviderID == 0 {
			err = fmt.Errorf("provider with id %v not found", i.ProviderId)
		} else if provider.URL != i.Url {
			err = fmt.Errorf("invalid url in provider %v", provider.URL)
		} else if provider.UploadURL != i.UploadUrl {
			err = fmt.Errorf("invalid upload url in provider %v", provider.UploadURL)
		}
	} else {
		provider, err = base.WharfClient.GetProvider("github", i.Url, i.UploadUrl, base.Token.TokenID)
		if err != nil || provider.ProviderID == 0 {
			provider, err = base.WharfClient.PostProvider(wharfapi.Provider{Name: "github", URL: i.Url, UploadURL: i.UploadUrl})
		}
	}
	fmt.Println("Provider from db: ", provider)
	return provider, nil
}

func (base GithubImporter) GetToken(i Import) (wharfapi.Token, error) {
	var token wharfapi.Token
	var err error

	if base.Provider.ProviderID == 0 {
		return token, fmt.Errorf("provider not found")
	}

	if i.TokenId != 0 {
		token, err = base.WharfClient.GetTokenById(i.TokenId)
		if err != nil {
			return token, err
		} else if token.TokenID == 0 {
			err = fmt.Errorf(fmt.Sprintf("Token with id %v not found", i.TokenId))
		} else if token.ProviderID != base.Provider.ProviderID {
			err = fmt.Errorf(fmt.Sprintf("Token with invalid provider id %v.", i.ProviderId))
		}
	} else {
		token, err = base.WharfClient.GetToken(i.Token, i.User)
		if err != nil || token.TokenID == 0 {
			token, err = base.WharfClient.PostToken(wharfapi.Token{Token: i.Token, UserName: i.User, ProviderID: base.Provider.ProviderID})
		}
	}

	fmt.Println("Token from db: ", token)
	return token, err
}

func (base GithubImporter) InitGithubConnection() (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: base.Token.Token})
	tc := oauth2.NewClient(base.Context, ts)
	client, err := github.NewEnterpriseClient(base.Provider.URL, base.Provider.UploadURL, tc)
	return client, err
}

func (base GithubImporter) GetBuildDefiniton(owner string, projectName string) string {
	fileContent, _, _, err := base.GithubClient.Repositories.GetContents(base.Context, owner, projectName, buildDefinitionFileName, nil)
	if err != nil {
		return ""
	}

	bodyString, err := b64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		return ""
	}

	return string(bodyString)
}

func (base GithubImporter) ImportProject(i Import) error {
	if i.ProjectId != 0 {
		project, err := base.WharfClient.GetProjectById(i.ProjectId)
		if err != nil {
			return err
		} else if project.ProjectID == 0 {
			return fmt.Errorf(fmt.Sprintf("Project with id %v not found.", i.ProjectId))
		}
		i.Project = project.Name
	}

	var repo *github.Repository
	var err error
	if i.Group != "" {
		repo, _, err = base.GithubClient.Repositories.Get(base.Context, i.Group, i.Project)
		if err != nil {
			return err
		} else if repo.GetName() != i.Project {
			return fmt.Errorf(fmt.Sprintf("Project with name %v not found.", i.Project))
		} else if repo.GetOwner().GetLogin() != i.Group {
			return fmt.Errorf(fmt.Sprintf("Unable to find project with name %v in organization or associeted with user %v.",
				i.Project, repo.GetOwner().GetLogin()))
		}
	} else {
		repos, _, err := base.GithubClient.Repositories.List(base.Context, "", nil)
		if err != nil {
			return err
		}

		for _, repository := range repos {
			if repository.GetName() == i.Project {
				repo = repository
			}
		}
	}

	return base.PutProject(repo)
}

func (base GithubImporter) PutProject(repo *github.Repository) error {
	buildDefinitionStr := base.GetBuildDefiniton(repo.GetOwner().GetLogin(), repo.GetName())

	project, err := base.WharfClient.PutProject(
		wharfapi.Project{
			Name:            repo.GetName(),
			TokenID:         base.Token.TokenID,
			GroupName:       repo.GetOwner().GetLogin(),
			BuildDefinition: buildDefinitionStr,
			Description:     repo.GetDescription(),
			ProviderID:      base.Provider.ProviderID})
	if err != nil {
		return err
	} else if project.ProjectID == 0 {
		return fmt.Errorf("unable to put project")
	}

	branches, _, err := base.GithubClient.Repositories.ListBranches(base.Context, project.GroupName, project.Name, nil)
	if err != nil {
		return err
	}
	for _, branch := range branches {
		_, err := base.WharfClient.PutBranch(
			wharfapi.Branch{
				Name:      branch.GetName(),
				ProjectID: project.ProjectID,
				Default:   branch.GetName() == repo.GetDefaultBranch(),
				TokenID:   base.Token.TokenID})
		if err != nil {
			break
		}
	}

	return err
}

func (base GithubImporter) Import(groupName string) error {
	repos, _, err := base.GithubClient.Repositories.List(base.Context, "", nil)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if groupName == "" || repo.GetOwner().GetLogin() == groupName {
			err = base.PutProject(repo)
			if err != nil {
				return err
			}
		}
	}

	return err
}
