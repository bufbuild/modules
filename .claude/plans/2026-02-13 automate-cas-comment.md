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
- Comment 1: `casdiff foo bar --format=markdown` (in the line where the `bar` hash first appeared)
- Comment 2: `casdiff bar baz --format=markdown` (in the line where the `baz` hash first appeared)

## Solution

Create a Go command "Comment PR CAS Diff" (`cmd/commentprcasdiff`) that automates this process:

### 1. Input & Detection
- Run in GitHub Actions context
- Read PR number from environment (`github.event.pull_request.number`)
- Read base and head refs (typically `main` and `fetch-modules`)
- Identify changed `state.json` files in the PR

### 2. Core Logic

**For each `modules/sync/<owner>/<repo>/state.json` that changed:**

1. **Read full JSON from both branches:**
   - Read `state.json` from base branch (e.g., `main`)
   - Read `state.json` from head branch (e.g., `fetch-modules`)
   - Parse both as JSON to get reference arrays

2. **Identify appended references:**
   - Since `state.json` is append-only, compare array lengths
   - Extract only the newly appended references from the head branch
   - Track line numbers in the diff for each new reference

3. **Detect digest transitions in appended portion:**
   - Get the last reference from base state (has the "baseline" digest)
   - Iterate through appended references
   - When digest changes from previous reference:
     - Record: `StateTransition{fromRef, toRef, fromDigest, toDigest, lineNumber}`
   - Track line number where the new digest first appears

**Example:**
```
Base state: [..., {name: "v1.0.0", digest: "aaa"}]
Head state: [..., {name: "v1.0.0", digest: "aaa"},
                  {name: "v1.1.0", digest: "aaa"},  // L267: Same digest, no comment
                  {name: "v1.2.0", digest: "bbb"},  // L269: Digest changed! Comment here
                  {name: "v1.3.0", digest: "ccc"}]  // L271: Digest changed! Comment here
```

**Important:** Don't rely on git diff `-` (subtraction) lines since the file is append-only. Instead, parse the complete JSON arrays and compare them structurally.

### 3. Run casdiff & Comment

For each `StateTransition`:
1. Navigate to the module's directory: `modules/sync/<owner>/<repo>`
2. Run: `casdiff <fromRef> <toRef> --format=markdown`
3. Capture stdout
4. Post as a PR review comment on the specific line where the new digest appears
   - Use GitHub API to post comment at the line number tracked in StateTransition
   - This creates an inline review comment, not a general PR comment

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

      - name: Run commentprcasdiff and comment
        run: go run ./cmd/commentprcasdiff
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
          BASE_REF: ${{ github.event.pull_request.base.sha }}
          HEAD_REF: ${{ github.event.pull_request.head.sha }}
```

## Implementation Structure

```
cmd/
  commentprcasdiff/
    main.go              # Entry point, orchestrates the workflow
    module_finder.go     # Find changed module state.json files
    state_analyzer.go    # Analyze state.json changes to find digest transitions
    casdiff_runner.go    # Execute casdiff commands (with parallelization)
    comment_poster.go    # Post review comments to GitHub PR on specific lines
```

## Key Components

### main.go
- Parse environment variables (PR_NUMBER, BASE_REF, HEAD_REF)
- Identify changed module state directories
- For each module, analyze digest transitions
- Run casdiff for each transition (in parallel)
- Post review comments on specific lines
- Log any errors

### module_finder.go
```go
// ChangedModuleStatesFromPR returns module state directories that changed in the PR
// Returns paths like "modules/sync/bufbuild/protovalidate"
func ChangedModuleStatesFromPR(prNumber, baseRef, headRef string) ([]string, error)
```

### state_analyzer.go
```go
type StateTransition struct {
    ModulePath string  // e.g., "modules/sync/bufbuild/protovalidate"
    FilePath   string  // e.g., "modules/sync/bufbuild/protovalidate/state.json"
    FromRef    string  // Last reference with old digest (e.g., "v1.1.0")
    ToRef      string  // First reference with new digest (e.g., "v1.2.0")
    FromDigest string  // Old digest
    ToDigest   string  // New digest
    LineNumber int     // Line in diff where new digest first appears
}

// GetStateFileTransitions reads state.json from base and head branches,
// compares the JSON arrays to find appended references, and detects digest transitions.
// baseRef and headRef are typically "main" and "fetch-modules" respectively.
func GetStateFileTransitions(filePath, baseRef, headRef string) ([]StateTransition, error)
```

### casdiff_runner.go
```go
type CASDiffResult struct {
    Transition StateTransition
    Output     string  // Markdown output from casdiff
    Error      error
}

// RunCASDiff executes casdiff command in the module directory
func RunCASDiff(transition StateTransition) CASDiffResult

// RunCASDiffParallel runs multiple casdiff commands concurrently
func RunCASDiffParallel(transitions []StateTransition) []CASDiffResult
```

### comment_poster.go
```go
type PRReviewComment struct {
    PRNumber   string
    FilePath   string  // File path in the PR (e.g., "modules/sync/bufbuild/protovalidate/state.json")
    LineNumber int     // Line number in the diff
    Body       string  // Comment body (casdiff output)
    CommitID   string  // Head commit SHA
}

// PostReviewComments posts review comments to specific lines in the PR diff.
// Uses GitHub API (via gh CLI or direct API) to create inline comments.
func PostReviewComments(comments ...PRReviewComment) []error
```

## Algorithm Details

### Finding Digest Transitions

Given a state.json file path, base ref, and head ref:

1. **Read and parse JSON from both branches:**
   ```go
   baseContent := git show baseRef:filePath
   headContent := git show headRef:filePath
   baseState := parseJSON(baseContent)  // {references: [...]}
   headState := parseJSON(headContent)  // {references: [...]}
   ```

2. **Identify appended references:**
   ```go
   baseCount := len(baseState.references)
   headCount := len(headState.references)
   appendedRefs := headState.references[baseCount:]  // Only new ones
   ```

3. **Find the baseline digest:**
   ```go
   var currentDigest string
   var currentRef string
   if baseCount > 0 {
       lastBaseRef := baseState.references[baseCount-1]
       currentDigest = lastBaseRef.digest
       currentRef = lastBaseRef.name
   }
   ```

4. **Detect transitions in appended references:**
   ```go
   var transitions []StateTransition
   for i, ref := range appendedRefs {
       if ref.digest != currentDigest {
           // Digest changed! Record transition
           transition := StateTransition{
               FromRef:    currentRef,
               ToRef:      ref.name,
               FromDigest: currentDigest,
               ToDigest:   ref.digest,
               LineNumber: calculateLineNumber(baseCount + i, headContent),
           }
           transitions = append(transitions, transition)

           // Update current for next iteration
           currentDigest = ref.digest
       }
       currentRef = ref.name
   }
   ```

5. **Calculate line numbers:**
   - Parse the diff to map array indices to line numbers in the diff
   - The line number should point to where `"digest": "new_hash"` appears

**Edge cases:**
- If base state is empty (new module), first reference becomes baseline (no comment)
- If no new references appended, return empty transitions
- If all appended references have same digest as baseline, return empty transitions

### Posting Comments on Specific Lines

Use GitHub's review comment API to post inline comments:

```bash
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments \
  -f body="<casdiff output>" \
  -f path="modules/sync/bufbuild/protovalidate/state.json" \
  -f commit_id="<head_sha>" \
  -F line=269
```

This creates a review comment attached to a specific line in the diff, similar to manual code review comments.

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
- Dry-run mode for testing
- Support running locally (not just in GitHub Actions)
