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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bufbuild/modules/internal/githubutil"
	"github.com/google/go-github/v64/github"
)

// prReviewComment represents a comment to be posted or patched on a specific line in a PR.
type prReviewComment struct {
	filePath   string // File path in the PR (e.g., "modules/sync/bufbuild/protovalidate/state.json")
	lineNumber int    // Line number in the diff
	body       string // Comment body (casdiff output)
}

// commentKey uniquely identifies a comment by file and line.
type commentKey struct {
	path string
	line int
}

// postReviewComments posts review comments to specific lines in the PR diff. If a bot comment
// already exists at the same file/line, it is updated instead of creating a duplicate.
func postReviewComments(ctx context.Context, prNumber int, gitCommitID string, comments ...prReviewComment) error {
	client := githubutil.NewClient(ctx)

	existingPRComments, err := listExistingBotComments(ctx, client, prNumber)
	if err != nil {
		return fmt.Errorf("list existing bot comments: %w", err)
	}

	var errsPosting []error
	for _, comment := range comments {
		key := commentKey{path: comment.filePath, line: comment.lineNumber}
		if prCommentID, alreadyPosted := existingPRComments[key]; alreadyPosted {
			if err := updateReviewComment(ctx, client, prCommentID, comment.body); err != nil {
				errsPosting = append(errsPosting, fmt.Errorf("updating existing comment in %v: %w", key, err))
			}
		} else {
			if err := postSingleReviewComment(ctx, client, prNumber, gitCommitID, comment); err != nil {
				errsPosting = append(errsPosting, fmt.Errorf("posting new comment in %v: %w", key, err))
			}
		}
	}
	return errors.Join(errsPosting...)
}

// listExistingBotComments returns a map of (path, line) → commentID for all comments left by
// github-actions[bot] on the PR. Sorted ascending by creation time so that when multiple bot
// comments exist at the same file+line, the highest-ID (most recently created) one wins.
func listExistingBotComments(ctx context.Context, client *githubutil.Client, prNumber int) (map[commentKey]int64, error) {
	const githubActionsBotUsername = "github-actions[bot]"
	result := make(map[commentKey]int64)
	opts := &github.PullRequestListCommentsOptions{
		Sort:        "created",
		Direction:   "asc",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		comments, resp, err := client.GitHub.PullRequests.ListComments(
			ctx,
			string(githubutil.GithubOwnerBufbuild),
			string(githubutil.GithubRepoModules),
			prNumber,
			opts,
		)
		if err != nil {
			return nil, fmt.Errorf("list PR comments: %w", err)
		}
		for _, comment := range comments {
			if comment.GetUser().GetLogin() == githubActionsBotUsername {
				key := commentKey{path: comment.GetPath(), line: comment.GetLine()}
				result[key] = comment.GetID()
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}

func postSingleReviewComment(ctx context.Context, client *githubutil.Client, prNumber int, gitCommitID string, comment prReviewComment) error {
	body := fmt.Sprintf("_[Posted at %s]_\n\n%s", time.Now().Format(time.RFC3339), comment.body)
	created, _, err := client.GitHub.PullRequests.CreateComment(
		ctx,
		string(githubutil.GithubOwnerBufbuild),
		string(githubutil.GithubRepoModules),
		prNumber,
		&github.PullRequestComment{
			CommitID: &gitCommitID,
			Path:     &comment.filePath,
			Line:     &comment.lineNumber,
			Body:     &body,
		},
	)
	if err != nil {
		return fmt.Errorf("create PR comment: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Posted comment: %s\n", created.GetHTMLURL())
	return nil
}

func updateReviewComment(ctx context.Context, client *githubutil.Client, prCommentID int64, body string) error {
	body = fmt.Sprintf("_[Updated at %s]_\n\n%s", time.Now().Format(time.RFC3339), body)
	updated, _, err := client.GitHub.PullRequests.EditComment(
		ctx,
		string(githubutil.GithubOwnerBufbuild),
		string(githubutil.GithubRepoModules),
		prCommentID,
		&github.PullRequestComment{
			Body: &body,
		},
	)
	if err != nil {
		return fmt.Errorf("edit PR comment: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Updated comment: %s\n", updated.GetHTMLURL())
	return nil
}
