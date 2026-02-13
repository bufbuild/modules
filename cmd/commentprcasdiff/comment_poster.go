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
	"fmt"
	"os"
	"os/exec"
)

// PRReviewComment represents a comment to post on a specific line in a PR.
type PRReviewComment struct {
	PRNumber   string
	FilePath   string // File path in the PR (e.g., "modules/sync/bufbuild/protovalidate/state.json")
	LineNumber int    // Line number in the diff
	Body       string // Comment body (casdiff output)
	CommitID   string // Head commit SHA
}

// PostReviewComments posts review comments to specific lines in the PR diff.
// Uses GitHub API (via gh CLI) to create inline comments.
func PostReviewComments(ctx context.Context, comments ...PRReviewComment) []error {
	var errors []error

	// Get repository information
	repo, err := getRepositoryInfo(ctx)
	if err != nil {
		for range comments {
			errors = append(errors, fmt.Errorf("get repository info: %w", err))
		}
		return errors
	}

	// Post each comment
	for _, comment := range comments {
		if err := postSingleReviewComment(ctx, repo, comment); err != nil {
			errors = append(errors, err)
		} else {
			errors = append(errors, nil)
		}
	}

	return errors
}

// getRepositoryInfo gets the owner/repo from the current git repository.
func getRepositoryInfo(ctx context.Context) (string, error) {
	// Use gh CLI to get repository info
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh repo view: %w", err)
	}
	return string(bytes.TrimSpace(output)), nil
}

// postSingleReviewComment posts a single review comment using GitHub API.
func postSingleReviewComment(ctx context.Context, repo string, comment PRReviewComment) error {
	// Prepare the API request body
	requestBody := map[string]interface{}{
		"body":      comment.Body,
		"commit_id": comment.CommitID,
		"path":      comment.FilePath,
		"line":      comment.LineNumber,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Use gh CLI to make the API request
	// gh api repos/{owner}/{repo}/pulls/{pr_number}/comments -X POST --input -
	cmd := exec.CommandContext(
		ctx,
		"gh", "api",
		fmt.Sprintf("repos/%s/pulls/%s/comments", repo, comment.PRNumber),
		"-X", "POST",
		"--input", "-",
	)

	cmd.Stdin = bytes.NewReader(requestJSON)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if GITHUB_TOKEN is set
		if os.Getenv("GITHUB_TOKEN") == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable not set")
		}
		return fmt.Errorf("gh api failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}
