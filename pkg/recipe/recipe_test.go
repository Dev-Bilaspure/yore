package recipe

import (
	"path/filepath"
	"strings"
	"testing"
)

const sample = `
# Team runbook
[deploy staging]
desc: build and ship to staging
cmd: ./deploy.sh --env {env} --tag {tag}
param: env=staging
tags: deploy, k8s

[port-forward]
cmd: kubectl port-forward svc/api 8080:80 -n {env}

[no command here]
desc: should be skipped
`

func TestParse(t *testing.T) {
	got, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("parsed %d recipes, want 2 (cmd-less skipped): %+v", len(got), got)
	}

	d := got[0]
	if d.Name != "deploy staging" || d.Description != "build and ship to staging" {
		t.Errorf("recipe[0] meta wrong: %+v", d)
	}
	if len(d.Params) != 2 || d.Params[0].Name != "env" || d.Params[1].Name != "tag" {
		t.Errorf("params = %+v, want env then tag (cmd order)", d.Params)
	}
	if d.Params[0].Default != "staging" {
		t.Errorf("env default = %q, want staging", d.Params[0].Default)
	}
	if len(d.Tags) != 2 || d.Tags[0] != "deploy" {
		t.Errorf("tags = %v", d.Tags)
	}
}

func TestParamsInOrderAndDedup(t *testing.T) {
	got := ParamsIn("a {x} b {y} c {x} d {z}")
	want := []string{"x", "y", "z"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ParamsIn = %v, want %v", got, want)
	}
}

func TestRender(t *testing.T) {
	r := Recipe{Command: "deploy --env {env} --tag {tag} --env {env}"}
	got := r.Render(map[string]string{"env": "prod", "tag": "v1"})
	want := "deploy --env prod --tag v1 --env prod"
	if got != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
	// Missing values are left as-is.
	if g := r.Render(map[string]string{"env": "prod"}); !strings.Contains(g, "{tag}") {
		t.Errorf("missing param should be left untouched, got %q", g)
	}
}

func TestFormatRoundTrip(t *testing.T) {
	in := Recipe{
		Name:        "seed db",
		Command:     "./seed.sh {env}",
		Description: "load fixtures",
		Params:      []Param{{Name: "env", Default: "dev"}},
		Tags:        []string{"db"},
	}
	got, err := Parse(strings.NewReader(Format(in)))
	if err != nil || len(got) != 1 {
		t.Fatalf("round-trip parse failed: %v / %d", err, len(got))
	}
	r := got[0]
	if r.Name != in.Name || r.Command != in.Command || r.Description != in.Description {
		t.Errorf("round-trip mismatch: %+v", r)
	}
	if len(r.Params) != 1 || r.Params[0].Default != "dev" {
		t.Errorf("param default lost in round-trip: %+v", r.Params)
	}
}

func TestMergeProjectPrecedence(t *testing.T) {
	global := []Recipe{{Name: "build", Command: "make"}, {Name: "test", Command: "go test"}}
	project := []Recipe{{Name: "build", Command: "./build.sh"}}
	merged := Merge(global, project)
	b, ok := Find(merged, "build")
	if !ok || b.Command != "./build.sh" {
		t.Errorf("project recipe should shadow global: %+v", b)
	}
	if len(merged) != 2 {
		t.Errorf("merged len = %d, want 2", len(merged))
	}
}

func TestLoadAndAppendFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectFile)
	if err := AppendToFile(path, Recipe{Name: "x", Command: "echo {a}"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendToFile(path, Recipe{Name: "y", Command: "ls"}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path, "project")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Source != "project" {
		t.Fatalf("loaded %+v", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	got, err := LoadFile(filepath.Join(t.TempDir(), "nope"), "global")
	if err != nil || got != nil {
		t.Errorf("missing file should be (nil,nil), got %v / %v", got, err)
	}
}
