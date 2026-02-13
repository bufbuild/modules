# Automate CAS Diff Comments on PRs

**Date:** 2026-02-13
**Branch:** `jfigueroa/automate-cas-comment`

## Context

The `modules` repository has a daily GitHub Action (`.github/workflows/fetch.yaml`) that:
- Fetches new git commits/releases from upstream repositories
- Stores proto files in CAS (content-addressable storage) format
- Updates `state.json` files with new references and their content digests
- Creates a PR (to branch `fetch-modules`) with these updates

Each module's state.json file (e.g., `modules/sync/bufbuild/protovalidate/state.json`) contains:
```json
{
  "references": [
    {
      "name": "v1.0.0",
      "digest": "ddff976f..."
    },
    {
      "name": "v1.1.0",
      "digest": "35b3d88f..."
    }
  ]
}
```

## Problem

When a digest changes (indicating actual proto content changed, not just a new tag pointing to same content), we currently manually:
1. Navigate to `modules/sync/<owner>/<repo>`
2. Run `casdiff <previous_ref> <new_ref> --format=markdown`
3. Comment the output on the PR

Example: If a PR adds references with digests `[foo, bar, baz]` where the previous latest was `foo`, we need:
- Comment 1: `casdiff <last_ref_with_foo> <first_ref_with_bar> --format=markdown`
- Comment 2: `casdiff <last_ref_with_bar> <first_ref_with_baz> --format=markdown`

## Solution

Create a Go command (`cmd/prdiff`) that automates this process:

### 1. Input & Detection
- Run in GitHub Actions context
- Read PR number from environment (`github.event.pull_request.number`)
- Fetch PR diff using `gh` CLI
- Parse diff to identify changed `state.json` files
- For each changed state.json, identify digest transitions

### 2. Core Logic

**For each `modules/sync/<owner>/<repo>/state.json` that changed:**

1. Parse the git diff to extract old and new state
2. Build a list of digest transitions:
   - Read old references (from diff `-` lines or base commit)
   - Read new references (from diff `+` lines or head commit)
   - Identify where digest changes occur
   - For each digest change, record: `(last_ref_with_old_digest, first_ref_with_new_digest)`

**Example:**
```
Old state: [..., {name: "v1.0.0", digest: "aaa"}]
New state: [..., {name: "v1.0.0", digest: "aaa"},
                {name: "v1.1.0", digest: "aaa"},  // Same digest, no comment needed
                {name: "v1.2.0", digest: "bbb"},  // Digest changed! Comment: casdiff v1.1.0 v1.2.0
                {name: "v1.3.0", digest: "ccc"}]  // Digest changed! Comment: casdiff v1.2.0 v1.3.0
```

### 3. Run casdiff & Comment

For each digest transition `(fromRef, toRef)`:
1. Navigate to the module's directory: `modules/sync/<owner>/<repo>`
2. Run: `casdiff <fromRef> <toRef> --format=markdown`
3. Capture stdout
4. Post as a PR comment using `gh pr comment <pr_number> --body <output>`

**Run casdiff calls in parallel** using goroutines for performance.

### 4. Error Handling

- If casdiff fails for a transition, log the error to action output (not as a comment)
- Continue processing other transitions
- At the end, if any failures occurred, log summary of all failures
- Don't fail the entire workflow due to individual casdiff failures

### 5. GitHub Actions Integration

Create `.github/workflows/auto-casdiff-comment.yaml`:

```yaml
name: Auto-comment PR with casdiff

on:
  pull_request:
    types: [opened, synchronize]
    branches:
      - fetch-modules

permissions:
  contents: read
  pull-requests: write

jobs:
  comment-changes:
    runs-on: ubuntu-latest
    if: github.repository == 'bufbuild/modules'
    steps:
      - name: Checkout repository
        uses: actions/checkout@v6
        with:
          fetch-depth: 0  # Need full history

      - name: Checkout PR head
        run: |
          git fetch origin pull/${{ github.event.pull_request.number }}/head:pr-head
          git checkout pr-head

      - name: Install Go
        uses: actions/setup-go@v6
        with:
          go-version: 1.24.x
          check-latest: true
          cache: true

      - name: Run prdiff and comment
        run: go run ./cmd/prdiff
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
          BASE_REF: ${{ github.event.pull_request.base.sha }}
          HEAD_REF: ${{ github.event.pull_request.head.sha }}
```

## Implementation Structure

```
cmd/
  prdiff/
    main.go              # Entry point, orchestrates the workflow
    diff_parser.go       # Parse git diff to extract state.json changes
    state_analyzer.go    # Analyze state.json changes to find digest transitions
    casdiff_runner.go    # Execute casdiff commands (with parallelization)
    comment_poster.go    # Post comments to GitHub PR
```

## Key Components

### main.go
- Parse environment variables (PR_NUMBER, BASE_REF, HEAD_REF)
- Get PR diff using `gh pr diff <pr_number>`
- Identify changed state.json files
- For each state.json, analyze digest transitions
- Run casdiff for each transition (in parallel)
- Post comments
- Log any errors

### diff_parser.go
```go
// ParsePRDiff extracts paths of changed state.json files from PR diff
func ParsePRDiff(prNumber string) ([]string, error)

// GetStateFileChanges reads old and new content of a state.json file
func GetStateFileChanges(filePath, baseRef, headRef string) (oldContent, newContent []byte, error)
```

### state_analyzer.go
```go
type DigestTransition struct {
    ModulePath string  // e.g., "modules/sync/bufbuild/protovalidate"
    FromRef    string  // Last reference with old digest
    ToRef      string  // First reference with new digest
    FromDigest string
    ToDigest   string
}

// AnalyzeDigestTransitions compares old and new state to find digest changes
func AnalyzeDigestTransitions(oldState, newState []byte, modulePath string) ([]DigestTransition, error)
```

### casdiff_runner.go
```go
type CasdiffResult struct {
    Transition DigestTransition
    Output     string
    Error      error
}

// RunCasdiff executes casdiff command in the module directory
func RunCasdiff(transition DigestTransition) CasdiffResult

// RunCasdiffParallel runs multiple casdiff commands concurrently
func RunCasdiffParallel(transitions []DigestTransition) []CasdiffResult
```

### comment_poster.go
```go
// PostComment posts a comment to the PR using gh CLI
func PostComment(prNumber string, body string) error

// PostAllComments posts all successful casdiff results as separate comments
func PostAllComments(prNumber string, results []CasdiffResult) []error
```

## Algorithm Details

### Finding Digest Transitions

Given old and new state.json references:

1. Parse both old and new references into lists
2. Find the last reference in the old state (this has the "current" digest)
3. Iterate through new references starting after the last old reference
4. Track the current digest
5. When digest changes:
   - Record transition: (last_ref_with_current_digest, current_ref)
   - Update current_digest
6. Return all transitions

**Edge cases:**
- If old state is empty, compare first ref against itself (no comment)
- If no new references added, return empty transitions
- If all new references have same digest as old, return empty transitions

## Testing Strategy

1. **Unit tests** for each component:
   - Test state parsing
   - Test digest transition detection with various scenarios
   - Test diff parsing

2. **Integration test** with test fixtures:
   - Create sample PR diffs
   - Create sample state.json files
   - Verify correct transitions are detected

3. **Manual testing:**
   - Create a test PR manually with known changes
   - Run the tool locally
   - Verify comments are posted correctly

## Success Criteria

1. When a PR is created by the fetch workflow with digest changes:
   - All digest transitions are detected correctly
   - casdiff is run for each transition
   - Comments are posted to the PR automatically

2. When a PR has no digest changes (only new refs with same digest):
   - No comments are posted

3. If casdiff fails for some transitions:
   - Successful transitions still get commented
   - Failures are logged in the action output

4. Performance:
   - Multiple casdiff commands run in parallel
   - Workflow completes in reasonable time (< 5 minutes for typical PR)

## Future Enhancements (Out of Scope)

- Update existing comments instead of posting new ones
- Add a comment summary at the top listing all modules changed
- Support for commenting on specific lines in the diff
- Dry-run mode for testing
