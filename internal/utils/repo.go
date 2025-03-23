package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

// RepoStatus contains information about a Git repository
type RepoStatus struct {
	IsValid       bool
	IsDirty       bool
	Branch        string
	ShortHash     string
	FullHash      string
	CommitMessage string
	CommitAuthor  string
	CommitDate    time.Time
}

// GetRepoStatus retrieves the current status of the Git repository
func GetRepoStatus() (RepoStatus, error) {
	status := RepoStatus{IsValid: false}

	// Open repository in the current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		return status, fmt.Errorf("failed to open git repository: %w", err)
	}

	status.IsValid = true

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return status, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Get branch name
	if head.Name().IsBranch() {
		status.Branch = head.Name().Short()
	} else {
		status.Branch = "detached-HEAD"
	}

	// Get commit hash
	hash := head.Hash()
	status.ShortHash = hash.String()[:7]
	status.FullHash = hash.String()

	// Get commit information
	commit, err := repo.CommitObject(hash)
	if err == nil {
		status.CommitMessage = commit.Message
		status.CommitAuthor = commit.Author.Name
		status.CommitDate = commit.Author.When
	}

	// Check if working tree is dirty
	worktree, err := repo.Worktree()
	if err != nil {
		return status, fmt.Errorf("failed to get worktree: %w", err)
	}

	wStatus, err := worktree.Status()
	if err != nil {
		return status, fmt.Errorf("failed to get worktree status: %w", err)
	}

	status.IsDirty = !wStatus.IsClean()

	return status, nil
}

// GetCommitDetails returns detailed information about the last commit
func GetCommitDetails() (string, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	// Format commit details
	details := fmt.Sprintf("commit %s\n", head.Hash())
	details += fmt.Sprintf("Author: %s <%s>\n", commit.Author.Name, commit.Author.Email)
	details += fmt.Sprintf("Date:   %s\n\n", commit.Author.When.Format(time.RFC1123Z))
	details += fmt.Sprintf("    %s\n", commit.Message)

	return details, nil
}

// GetUncommittedChanges returns the diff of uncommitted changes
func GetUncommittedChanges() (string, error) {
	// We'll execute git diff command for simplicity
	// While it's possible to do this with go-git, the formatting would be complex
	cmd := exec.Command("git", "diff")
	var output strings.Builder
	cmd.Stdout = &output

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run git diff: %w", err)
	}

	diff := output.String()
	if diff == "" {
		diff = "[No uncommitted changes]"
	}

	return diff, nil
}

// SanitizeBranchName replaces invalid characters in a branch name
func SanitizeBranchName(name string) string {
	// https://git-scm.com/docs/git-check-ref-format
	return strings.ReplaceAll(name, "/", "-")
}
