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
	"os"
	"os/exec"
)

// prReviewComment represents a comment to post on a specific line in a PR.
type prReviewComment struct {
	prNumber   string
	filePath   string // File path in the PR (e.g., "modules/sync/bufbuild/protovalidate/state.json")
	lineNumber int    // Line number in the diff
	body       string // Comment body (casdiff output)
	commitID   string // Head commit SHA
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
func postReviewComments(ctx context.Context, comments ...prReviewComment) []error {
	errs := make([]error, len(comments))

	repo, err := getRepositoryInfo(ctx)
	if err != nil {
		for i := range errs {
			errs[i] = fmt.Errorf("get repository info: %w", err)
		}
		return errs
	}

	// Fetch existing bot comments once, keyed by (path, line).
	existing, err := listExistingBotComments(ctx, repo, comments[0].prNumber)
	if err != nil {
		// Non-fatal: fall back to always creating new comments.
		fmt.Fprintf(os.Stderr, "Warning: could not fetch existing comments, will create new ones: %v\n", err)
		existing = map[commentKey]int64{}
	}

	for i, comment := range comments {
		key := commentKey{path: comment.filePath, line: comment.lineNumber}
		if existingID, ok := existing[key]; ok {
			errs[i] = updateReviewComment(ctx, repo, existingID, comment.body)
		} else {
			errs[i] = postSingleReviewComment(ctx, repo, comment)
		}
	}

	return errs
}

// listExistingBotComments returns a map of (path, line) → comment ID for all comments left by
// github-actions[bot] on the PR.
func listExistingBotComments(ctx context.Context, repo string, prNumber string) (map[commentKey]int64, error) {
	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"gh", "api",
		fmt.Sprintf("repos/%s/pulls/%s/comments?sort=created&direction=asc", repo, prNumber),
		"--paginate",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh api list comments: %w (stderr: %s)", err, stderr.String())
	}

	var rawComments []existingReviewComment
	if err := json.Unmarshal(stdout.Bytes(), &rawComments); err != nil {
		return nil, fmt.Errorf("parse comments response: %w", err)
	}

	result := make(map[commentKey]int64)
	for _, c := range rawComments {
		if c.User.Login == "github-actions[bot]" {
			result[commentKey{path: c.Path, line: c.Line}] = c.ID
		}
	}
	return result, nil
}

// updateReviewComment updates the body of an existing review comment.
func updateReviewComment(ctx context.Context, repo string, commentID int64, body string) error {
	requestBody := map[string]any{"body": body}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"gh", "api",
		fmt.Sprintf("repos/%s/pulls/comments/%d", repo, commentID),
		"-X", "PATCH",
		"--input", "-",
	)

	cmd.Stdin = bytes.NewReader(requestJSON)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh api patch comment: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// getRepositoryInfo gets the owner/repo from the current git repository.
func getRepositoryInfo(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh repo view: %w", err)
	}
	return string(bytes.TrimSpace(output)), nil
}

// postSingleReviewComment posts a single review comment using GitHub API.
func postSingleReviewComment(ctx context.Context, repo string, comment prReviewComment) error {
	requestBody := map[string]any{
		"body":      comment.body,
		"commit_id": comment.commitID,
		"path":      comment.filePath,
		"line":      comment.lineNumber,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"gh", "api",
		fmt.Sprintf("repos/%s/pulls/%s/comments", repo, comment.prNumber),
		"-X", "POST",
		"--input", "-",
	)

	cmd.Stdin = bytes.NewReader(requestJSON)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if os.Getenv("GITHUB_TOKEN") == "" {
			return errors.New("GITHUB_TOKEN environment variable not set")
		}
		return fmt.Errorf("gh api post comment: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}
