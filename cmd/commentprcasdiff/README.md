# commentprcasdiff

Automates posting CAS diff comments on PRs when module digest changes are detected.

## What it does

This tool runs in GitHub Actions when a PR is created against the `fetch-modules` branch. It:

1. Identifies which `state.json` files changed in the PR
2. Compares old and new state to find digest transitions (hash changes)
3. Runs `casdiff` for each transition to show what changed
4. Posts the output as inline review comments at the line where each new digest appears

## How it works

### State Analysis

Each module's `state.json` file is append-only and contains references with their content digests:

```json
{
  "references": [
    {"name": "v1.0.0", "digest": "aaa..."},
    {"name": "v1.1.0", "digest": "bbb..."}
  ]
}
```

The tool:
- Reads the full JSON from both the base branch (`main`) and head branch (`fetch-modules`)
- Identifies newly appended references
- Detects when the digest changes between consecutive references
- For each digest change, runs: `casdiff <old_ref> <new_ref> --format=markdown`

### Comment Posting

Comments are posted as PR review comments on the specific line where the new digest first appears in the diff, similar to manual code review comments.

Example: If digest changes from `aaa` to `bbb` at reference `v1.1.0`, a comment is posted at the line containing `"digest": "bbb"` in the state.json diff.

## Local Testing

To test the command locally:

```bash
# Set required environment variables
export PR_NUMBER=1234
export BASE_REF=main
export HEAD_REF=fetch-modules
export GITHUB_TOKEN=<your_token>

# Run the command
go run ./cmd/commentprcasdiff
```

**Note:** The command expects to be run from the repository root and requires:
- Git repository with the specified refs
- GitHub CLI (`gh`) installed and authenticated
- Access to post PR comments via GitHub API

## Architecture

- **main.go**: Entry point, orchestrates the workflow
- **module_finder.go**: Finds changed `state.json` files using git diff
- **state_analyzer.go**: Compares JSON arrays to detect digest transitions
- **casdiff_runner.go**: Executes casdiff commands in parallel
- **comment_poster.go**: Posts review comments via GitHub API

## Error Handling

- If casdiff fails for a transition, the error is logged but doesn't stop the workflow
- Successful transitions still get commented
- All failures are summarized at the end
