package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/indihood/yore/pkg/recipe"
)

func TestSaveRunRecipesFlow(t *testing.T) {
	withTempData(t)

	// Save a parameterized recipe to the personal store.
	_, stderr, code := run(t, "save", "--desc", "forward staging", "pf",
		"--", "kubectl", "port-forward", "svc/api", "8080:80", "-n", "{ns}")
	if code != 0 {
		t.Fatalf("save exit %d: %s", code, stderr)
	}

	// It should appear in the recipe list.
	out, _, _ := run(t, "recipes")
	if !strings.Contains(out, "pf") || !strings.Contains(out, "forward staging") {
		t.Errorf("recipes list missing saved recipe:\n%s", out)
	}

	// run --print --yes renders with (empty) defaults, no prompt, no exec.
	rendered, _, code := run(t, "run", "--print", "--yes", "pf")
	if code != 0 {
		t.Fatalf("run --print exit %d", code)
	}
	if !strings.Contains(rendered, "kubectl port-forward svc/api 8080:80 -n") {
		t.Errorf("rendered command unexpected: %q", rendered)
	}

	// Unknown recipe errors.
	if _, _, code := run(t, "run", "does-not-exist"); code != 1 {
		t.Errorf("running unknown recipe should exit 1, got %d", code)
	}
}

// Flags must work whether they come before OR after the recipe name; the
// natural `yore run <name> --print` form must not accidentally execute.
func TestRunFlagsAfterName(t *testing.T) {
	withTempData(t)
	if _, _, c := run(t, "save", "echoer", "--", "echo", "{msg}"); c != 0 {
		t.Fatalf("save exit %d", c)
	}
	for _, args := range [][]string{
		{"run", "echoer", "--print", "--yes"},
		{"run", "--print", "--yes", "echoer"},
		{"run", "--print", "echoer", "--yes"},
	} {
		out, _, code := run(t, args...)
		if code != 0 {
			t.Fatalf("%v exit %d", args, code)
		}
		if !strings.Contains(out, "echo") {
			t.Errorf("%v: expected printed command, got %q", args, out)
		}
	}
}

func TestPromptParams(t *testing.T) {
	params := []recipe.Param{
		{Name: "env", Default: "staging"},
		{Name: "tag"},
	}
	// User accepts the default for env (blank line) and types v2 for tag.
	in := strings.NewReader("\nv2\n")
	var prompts bytes.Buffer
	got := promptParams(in, &prompts, params, false)

	if got["env"] != "staging" {
		t.Errorf("env = %q, want default staging", got["env"])
	}
	if got["tag"] != "v2" {
		t.Errorf("tag = %q, want v2", got["tag"])
	}
	if !strings.Contains(prompts.String(), "env [staging]:") {
		t.Errorf("expected default shown in prompt, got %q", prompts.String())
	}
}

func TestPromptParamsAssumeYes(t *testing.T) {
	params := []recipe.Param{{Name: "env", Default: "dev"}, {Name: "x"}}
	got := promptParams(strings.NewReader(""), &bytes.Buffer{}, params, true)
	if got["env"] != "dev" || got["x"] != "" {
		t.Errorf("assumeYes values = %v, want env=dev x=''", got)
	}
}

func TestSaveLastFromStore(t *testing.T) {
	withTempData(t)
	// Record a command, then save it as a recipe via --last.
	if _, _, c := run(t, "record", "--command", "terraform apply -auto-approve", "--exit", "0"); c != 0 {
		t.Fatalf("record exit %d", c)
	}
	out, stderr, code := run(t, "save", "--last", "tf-apply")
	if code != 0 {
		t.Fatalf("save --last exit %d: %s", code, stderr)
	}
	if !strings.Contains(out, "tf-apply") {
		t.Errorf("save --last output: %q", out)
	}
	rendered, _, _ := run(t, "run", "--print", "--yes", "tf-apply")
	if !strings.Contains(rendered, "terraform apply -auto-approve") {
		t.Errorf("recipe from --last wrong: %q", rendered)
	}
}
