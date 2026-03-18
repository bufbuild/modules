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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// prReviewComment represents a comment to be posted or patched on a specific line in a PR.
type prReviewComment struct {
	filePath   string // File path in the PR (e.g., "modules/sync/bufbuild/protovalidate/state.json")
	lineNumber int    // Line number in the diff
	body       string // Comment body (casdiff output)
}

// existingReviewComment is a comment already posted on the PR.
type existingReviewComment struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
	Line int    `json:"line"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// commentKey uniquely identifies a comment by file and line.
type commentKey struct {
	path string
	line int
}

// postReviewComments posts review comments to specific lines in the PR diff. Uses GitHub API (via
// gh CLI) to create inline comments. If a bot comment already exists at the same file/line, it is
// updated instead of creating a duplicate.
func postReviewComments(ctx context.Context, prNumber uint, gitCommitID string, comments ...prReviewComment) error {
	gitRepoOwnerAndName, err := getGitRepositoryOwnerAndName(ctx)
	if err != nil {
		return fmt.Errorf("get repository owner/name: %w", err)
	}

	// Fetch existing pr comments bot comments, keyed by (path, line).
	existingPRComments, err := listExistingBotComments(ctx, gitRepoOwnerAndName, prNumber)
	if err != nil {
		return fmt.Errorf("list existing bot comments: %w", err)
	}

	var errsPosting []error
	for _, comment := range comments {
		key := commentKey{path: comment.filePath, line: comment.lineNumber}
		if prCommentID, alreadyPosted := existingPRComments[key]; alreadyPosted {
			if err := updateReviewComment(ctx, gitRepoOwnerAndName, prCommentID, comment.body); err != nil {
				errsPosting = append(errsPosting, fmt.Errorf("updating existing comment in %v: %w", key, err))
			}
		} else {
			if err := postSingleReviewComment(ctx, gitRepoOwnerAndName, prNumber, gitCommitID, comment); err != nil {
				errsPosting = append(errsPosting, fmt.Errorf("posting new comment in %v: %w", key, err))
			}
		}
	}
	if len(errsPosting) > 0 {
		return errors.Join(errsPosting...)
	}
	return nil
}

// getGitRepositoryOwnerAndName gets the owner/repo from the current git repository.
func getGitRepositoryOwnerAndName(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh repo view: %w", err)
	}
	return string(bytes.TrimSpace(output)), nil
}

// listExistingBotComments returns a map of (path, line) → commentID for all comments left by
// github-actions[bot] on the PR.
func listExistingBotComments(ctx context.Context, repo string, prNumber uint) (map[commentKey]int64, error) {
	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"gh", "api",
		fmt.Sprintf(
			"repos/%s/pulls/%d/comments?"+
				"sort=created&direction=asc", // determinism: always patch the same comment id
			repo,
			prNumber,
		),
		"--paginate",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh api list comments: %w (stderr: %s)", err, stderr.String())
	}

	var existingComments []existingReviewComment
	respBytes := stdout.Bytes()
	if err := json.Unmarshal(respBytes, &existingComments); err != nil {
		return nil, fmt.Errorf("parse comments response: %s: %w", string(respBytes), err)
	}

	const githubActionsBotUsername = "github-actions[bot]"
	result := make(map[commentKey]int64)
	for _, comment := range existingComments {
		if comment.User.Login == githubActionsBotUsername {
			result[commentKey{path: comment.Path, line: comment.Line}] = comment.ID
		}
	}
	return result, nil
}

func postSingleReviewComment(ctx context.Context, repoOwnerAndName string, prNumber uint, gitCommitID string, comment prReviewComment) error {
	commentBody := fmt.Sprintf("_[Posted at %s]_\n\n%s", time.Now().Format(time.RFC3339), comment.body)
	requestBody := map[string]any{
		"commit_id": gitCommitID,
		"path":      comment.filePath,
		"line":      comment.lineNumber,
		"body":      commentBody,
	}
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"gh", "api",
		fmt.Sprintf("repos/%s/pulls/%d/comments", repoOwnerAndName, prNumber),
		"-X", "POST",
		"--input", "-",
	)
	cmd.Stdin = bytes.NewReader(requestJSON)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh api post comment: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

func updateReviewComment(ctx context.Context, repoOwnerAndName string, prCommentID int64, body string) error {
	commentBody := fmt.Sprintf("_[Updated at %s]_\n\n%s", time.Now().Format(time.RFC3339), body)
	requestJSON, err := json.Marshal(map[string]any{"body": commentBody})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"gh", "api",
		fmt.Sprintf("repos/%s/pulls/comments/%d", repoOwnerAndName, prCommentID),
		"-X", "PATCH",
		"--input", "-",
	)
	cmd.Stdin = bytes.NewReader(requestJSON)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh api patch comment: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
