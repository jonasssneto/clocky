package ci

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCoverageTargetStopsWhenTestsFail(t *testing.T) {
	command := exec.CommandContext(context.Background(), "make", "-s", "test-coverage", "GO=false")
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("coverage target succeeded after test failure:\n%s", output)
	}
	if strings.Contains(string(output), "meets the") {
		t.Fatalf("coverage target reported success after test failure:\n%s", output)
	}
}

type workflow struct {
	On          map[string]any    `yaml:"on"`
	Permissions map[string]string `yaml:"permissions"`
	Jobs        map[string]job    `yaml:"jobs"`
}

type job struct {
	Needs       []string          `yaml:"needs"`
	If          string            `yaml:"if"`
	Permissions map[string]string `yaml:"permissions"`
	Steps       []step            `yaml:"steps"`
}

type step struct {
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

func TestBuildWorkflowRequiresQualityGatesBeforePublishing(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "build.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config workflow
	if err := yaml.Unmarshal(contents, &config); err != nil {
		t.Fatalf("parse build workflow: %v", err)
	}
	if _, ok := config.On["pull_request"]; !ok {
		t.Fatal("build workflow is not triggered by pull requests")
	}
	if _, ok := config.On["issue_comment"]; !ok {
		t.Fatal("build workflow is not triggered by PR comments")
	}
	if config.Permissions["contents"] != "read" {
		t.Fatal("workflow-wide contents permission must remain read-only")
	}

	lintJob, ok := config.Jobs["lint"]
	if !ok || !jobUses(lintJob, "golangci/golangci-lint-action@") {
		t.Fatal("lint job does not run golangci-lint")
	}
	testJob, ok := config.Jobs["test"]
	if !ok || !jobRuns(testJob, "make test-coverage") || !jobRuns(testJob, "go vet ./...") {
		t.Fatal("test job does not enforce coverage and go vet")
	}
	buildJob, ok := config.Jobs["build"]
	if !ok || !sameSet(buildJob.Needs, []string{"lint", "test"}) {
		t.Fatalf("build dependencies = %v, want lint and test", buildJob.Needs)
	}

	development, ok := config.Jobs["development-release"]
	if !ok {
		t.Fatal("development release job is missing")
	}
	if !sameSet(development.Needs, []string{"lint", "test", "build"}) {
		t.Fatalf("development release dependencies = %v", development.Needs)
	}
	if !strings.Contains(development.If, "github.event_name == 'issue_comment'") ||
		!strings.Contains(development.If, "github.event.comment.body == '/release-dev'") ||
		!strings.Contains(development.If, "github.event.issue.pull_request") ||
		!strings.Contains(development.If, "github.event.comment.author_association") {
		t.Fatalf("development release condition is unsafe: %q", development.If)
	}
	if development.Permissions["contents"] != "write" {
		t.Fatal("development release cannot publish without contents: write")
	}
	if !jobHasPrereleaseStep(development) {
		t.Fatal("development release is not marked as a prerelease")
	}
}

func jobUses(job job, prefix string) bool {
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, prefix) {
			return true
		}
	}
	return false
}

func jobRuns(job job, command string) bool {
	for _, step := range job.Steps {
		if strings.Contains(step.Run, command) {
			return true
		}
	}
	return false
}

func jobHasPrereleaseStep(job job) bool {
	for _, step := range job.Steps {
		prerelease, ok := step.With["prerelease"].(bool)
		if strings.HasPrefix(step.Uses, "softprops/action-gh-release@") && ok && prerelease {
			return true
		}
	}
	return false
}

func sameSet(left, right []string) bool {
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}
