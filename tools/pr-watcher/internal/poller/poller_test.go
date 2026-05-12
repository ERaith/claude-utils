package poller

import (
	"reflect"
	"testing"
)

func TestBuildGitHubArgs(t *testing.T) {
	got := BuildGitHubArgs("owner/name")
	want := []string{
		"pr", "list",
		"--repo", "owner/name",
		"--state", "open",
		"--json", "number,headRefOid,updatedAt,title,url",
		"--limit", "100",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildGitHubArgs:\n got %v\nwant %v", got, want)
	}
}

func TestBuildGitLabArgs(t *testing.T) {
	got := BuildGitLabArgs("group/proj")
	want := []string{
		"mr", "list",
		"-R", "group/proj",
		"--opened",
		"-F", "json",
		"--per-page", "100",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildGitLabArgs:\n got %v\nwant %v", got, want)
	}
}
