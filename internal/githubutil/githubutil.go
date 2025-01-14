// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package githubutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/google/go-github/v64/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	GithubOwnerBufbuild GithubOwner = "bufbuild"
	GithubRepoModules   GithubRepo  = "modules"

	githubPerPage = 50
)

var ErrNotFound = errors.New("release not found")

type GithubOwner string
type GithubRepo string
type Client struct {
	GitHub *github.Client
}

// NewClient returns a new HTTP client which can be used to perform actions on GitHub releases.
func NewClient(ctx context.Context) *Client {
	var httpClient *http.Client
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	} else {
		client := retryablehttp.NewClient()
		client.Logger = nil
		httpClient = client.StandardClient()
	}
	return &Client{
		GitHub: github.NewClient(httpClient),
	}
}

// CreateRelease creates a new GitHub release under the owner and repo.
func (c *Client) CreateRelease(
	ctx context.Context,
	owner GithubOwner,
	repo GithubRepo,
	release *github.RepositoryRelease,
) (*github.RepositoryRelease, error) {
	repositoryRelease, _, err := c.GitHub.Repositories.CreateRelease(ctx, string(owner), string(repo), release)
	if err != nil {
		return nil, err
	}
	return repositoryRelease, nil
}

// EditRelease performs an update of editable properties of a release (i.e. marking it not as a draft).
func (c *Client) EditRelease(
	ctx context.Context,
	owner GithubOwner,
	repo GithubRepo,
	releaseID int64,
	releaseChanges *github.RepositoryRelease,
) (*github.RepositoryRelease, error) {
	release, _, err := c.GitHub.Repositories.EditRelease(ctx, string(owner), string(repo), releaseID, releaseChanges)
	if err != nil {
		return nil, err
	}
	return release, nil
}

// UploadReleaseAsset uploads the specified file to the release.
func (c *Client) UploadReleaseAsset(
	ctx context.Context,
	owner GithubOwner,
	repo GithubRepo,
	releaseID int64,
	filename string,
) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, _, err = c.GitHub.Repositories.UploadReleaseAsset(
		ctx,
		string(owner),
		string(repo),
		releaseID,
		&github.UploadOptions{Name: filepath.Base(filename)},
		file)
	return err
}

// GetLatestRelease returns information about the latest GitHub release for the given org and repo (i.e. 'bufbuild', 'modules').
// If no release is found, returns ErrReleaseNotFound.
func (c *Client) GetLatestRelease(
	ctx context.Context,
	owner GithubOwner,
	repo GithubRepo,
) (*github.RepositoryRelease, error) {
	repositoryRelease, resp, err := c.GitHub.Repositories.GetLatestRelease(ctx, string(owner), string(repo))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// no latest release
			return nil, ErrNotFound
		}
		return nil, err
	}
	return repositoryRelease, nil
}

// GetReleaseByTag returns information about a given release (by tag name).
func (c *Client) GetReleaseByTag(
	ctx context.Context,
	owner GithubOwner,
	repo GithubRepo,
	tag string,
) (*github.RepositoryRelease, error) {
	repositoryRelease, resp, err := c.GitHub.Repositories.GetReleaseByTag(ctx, string(owner), string(repo), tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// no latest release
			return nil, ErrNotFound
		}
		return nil, err
	}
	return repositoryRelease, nil
}

// AllReleaseTagNames gets all release tag names for the repository.
func (c *Client) AllReleaseTagNames(
	ctx context.Context,
	owner string,
	repository string,
) ([]string, error) {
	var tagNames []string
	nextPage := 0
	for {
		repositoryReleases, response, err := c.GitHub.Repositories.ListReleases(
			ctx,
			owner,
			repository,
			&github.ListOptions{
				Page:    nextPage,
				PerPage: githubPerPage,
			},
		)
		if err != nil {
			return nil, err
		}
		for _, repositoryRelease := range repositoryReleases {
			if tagName := repositoryRelease.TagName; tagName != nil {
				tagNames = append(tagNames, *tagName)
			}
		}
		nextPage = response.NextPage
		if nextPage == 0 {
			return tagNames, nil
		}
	}
}

// DownloadReleaseState loads the state.json file from the specified GitHub release.
func (c *Client) DownloadReleaseState(
	ctx context.Context,
	release *github.RepositoryRelease,
) (*statev1alpha1.GlobalState, error) {
	manifestBytes, _, err := c.downloadAsset(ctx, release, bufstate.ModStateFileName)
	if err != nil {
		return nil, err
	}
	var moduleReleases statev1alpha1.GlobalState
	if err := protojson.Unmarshal(manifestBytes, &moduleReleases); err != nil {
		return nil, err
	}
	return &moduleReleases, nil
}

// downloadAsset uses the GitHub API to download the asset with the given name from the release.
// If the asset isn't found, returns ErrNotFound.
func (c *Client) downloadAsset(
	ctx context.Context,
	release *github.RepositoryRelease,
	assetName string,
) ([]byte, time.Time, error) {
	var assetID int64
	var assetLastModified time.Time
	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			assetID = asset.GetID()
			assetLastModified = asset.GetUpdatedAt().Time
			break
		}
	}
	if assetID == 0 {
		return nil, time.Time{}, ErrNotFound
	}
	owner, repo, err := getOwnerRepoFromReleaseURL(release.GetURL())
	if err != nil {
		return nil, time.Time{}, err
	}
	releaseAsset, _, err := c.GitHub.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, http.DefaultClient)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer func() {
		if err := releaseAsset.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "close: %v", err)
		}
	}()
	contents, err := io.ReadAll(releaseAsset)
	if err != nil {
		return nil, time.Time{}, err
	}
	return contents, assetLastModified, nil
}

// getOwnerRepoFromReleaseURL parses a URL in the format:
//
//	https://api.github.com/repos/{owner}/{repo}/releases/{release_id}
//
// into '{owner}' and '{repo}'. If not in the expected format, it will return a non-nil error.
func getOwnerRepoFromReleaseURL(url string) (string, string, error) {
	_, ownerRepo, found := strings.Cut(url, "/repos/")
	if !found {
		return "", "", fmt.Errorf("unsupported release URL format - no /repos/ found: %q", url)
	}
	ownerRepo, _, found = strings.Cut(ownerRepo, "/releases/")
	if !found {
		return "", "", fmt.Errorf("unsupported release URL format - no /releases/ found: %q", url)
	}
	owner, repo, found := strings.Cut(ownerRepo, "/")
	if !found {
		return "", "", fmt.Errorf("unsupported release URL format no owner/repo found: %q", url)
	}
	return owner, repo, nil
}
