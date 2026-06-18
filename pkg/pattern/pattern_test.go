package pattern

import (
	"strings"
	"testing"
	"time"
)

var ref = time.Unix(1_700_000_000, 0)

func obs(cmd string) Observation { return Observation{Command: cmd, Dir: "/proj", Time: ref} }

func only(t *testing.T, in []Observation) Candidate {
	t.Helper()
	got := Detect(in, Options{MinCount: 3, Now: ref})
	if len(got) != 1 {
		t.Fatalf("want exactly 1 candidate, got %d: %+v", len(got), got)
	}
	return got[0]
}

func TestTokenizeQuoting(t *testing.T) {
	toks := tokenize(`echo "a b" 'c' d\ e`)
	want := []string{"echo", "a b", "c", "d e"}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i].text != w {
			t.Errorf("token[%d]=%q, want %q", i, toks[i].text, w)
		}
	}
}

func TestFlagParameterization(t *testing.T) {
	c := only(t, []Observation{
		obs("./deploy.sh --env staging --tag v2.1.0"),
		obs("./deploy.sh --env prod --tag v2.3.0"),
		obs("./deploy.sh --env staging --tag v2.4.0"),
	})
	if c.Template != "./deploy.sh --env {env} --tag {tag}" {
		t.Errorf("template = %q", c.Template)
	}
	if c.Name != "deploy" {
		t.Errorf("name = %q, want deploy", c.Name)
	}
	if c.Count != 3 {
		t.Errorf("count = %d, want 3", c.Count)
	}
	if len(c.Params) != 2 || c.Params[0].Name != "env" || c.Params[1].Name != "tag" {
		t.Fatalf("params = %+v", c.Params)
	}
	if strings.Join(c.Params[0].Values, ",") != "staging,prod" {
		t.Errorf("env evidence = %v, want [staging prod]", c.Params[0].Values)
	}
}

func TestConstantFlagStaysLiteral(t *testing.T) {
	c := only(t, []Observation{
		obs("kubectl -n staging logs -f deploy/api"),
		obs("kubectl -n prod logs -f deploy/api"),
		obs("kubectl -n staging logs -f deploy/api"),
	})
	if c.Template != "kubectl -n {n} logs -f deploy/api" {
		t.Errorf("template = %q", c.Template)
	}
	if c.Name != "kubectl-logs" {
		t.Errorf("name = %q, want kubectl-logs", c.Name)
	}
	if len(c.Params) != 1 || c.Params[0].Name != "n" {
		t.Errorf("params = %+v, want just {n}", c.Params)
	}
}

func TestBooleanFlagZeroBlankCandidate(t *testing.T) {
	c := only(t, []Observation{
		obs("docker compose up -d"),
		obs("docker compose up -d"),
		obs("docker compose up -d"),
	})
	if c.Template != "docker compose up -d" || len(c.Params) != 0 {
		t.Errorf("expected literal zero-blank candidate, got %q params=%+v", c.Template, c.Params)
	}
	if c.Name != "docker-compose" {
		t.Errorf("name = %q, want docker-compose", c.Name)
	}
}

func TestPositionalValueParameterized(t *testing.T) {
	c := only(t, []Observation{
		obs("process /data/jan.csv"),
		obs("process /data/feb.csv"),
		obs("process /data/mar.csv"),
	})
	if c.Template != "process {arg1}" {
		t.Errorf("template = %q, want process {arg1}", c.Template)
	}
}

func TestParamNameSanitization(t *testing.T) {
	c := only(t, []Observation{
		obs("tool --dry-run a -n 1"),
		obs("tool --dry-run b -n 2"),
		obs("tool --dry-run a -n 3"),
	})
	// "--dry-run" -> {dry_run}; "-n" -> {n}
	if c.Template != "tool --dry-run {dry_run} -n {n}" {
		t.Errorf("template = %q", c.Template)
	}
}

func TestBelowThreshold(t *testing.T) {
	got := Detect([]Observation{
		obs("./deploy.sh --env staging --tag v2.1.0"),
		obs("./deploy.sh --env prod --tag v2.3.0"),
	}, Options{MinCount: 3, Now: ref})
	if len(got) != 0 {
		t.Errorf("below threshold should yield nothing, got %+v", got)
	}
}

func TestRecencyFilter(t *testing.T) {
	old := Observation{Command: "./deploy.sh --env prod --tag v1", Dir: "/p", Time: ref.Add(-60 * 24 * time.Hour)}
	in := []Observation{old, old, old, old, old}
	got := Detect(in, Options{MinCount: 3, Now: ref, RecentWithin: 30 * 24 * time.Hour})
	if len(got) != 0 {
		t.Errorf("stale commands should be filtered out, got %+v", got)
	}
}

func TestTrivialAndAliasesSkipped(t *testing.T) {
	for _, cmd := range []string{"ls -la", "glog", "git status", "cd .."} {
		in := []Observation{obs(cmd), obs(cmd), obs(cmd), obs(cmd), obs(cmd)}
		if got := Detect(in, Options{MinCount: 3, Now: ref}); len(got) != 0 {
			t.Errorf("%q should not be suggested, got %+v", cmd, got)
		}
	}
}

func TestWordPositionalsDoNotGroup(t *testing.T) {
	// git checkout <branch> — branches are word-shaped positionals, so they stay
	// in separate groups and none reaches the threshold (documented limitation).
	got := Detect([]Observation{
		obs("git checkout main"),
		obs("git checkout dev"),
		obs("git checkout main"),
		obs("git checkout feature"),
	}, Options{MinCount: 3, Now: ref})
	if len(got) != 0 {
		t.Errorf("word positionals shouldn't parameterize, got %+v", got)
	}
}

func TestKeyStableAcrossValuesAndEvolution(t *testing.T) {
	// Same command shape with different values → same key (so one dismissal
	// covers the whole family).
	a := only(t, []Observation{
		obs("./deploy.sh --env staging --tag v1"),
		obs("./deploy.sh --env prod --tag v1"),
		obs("./deploy.sh --env staging --tag v1"),
	})
	// Later: --tag now also varies (skeleton evolves to two blanks), but the
	// structural signature — and thus the key — is unchanged.
	b := only(t, []Observation{
		obs("./deploy.sh --env staging --tag v1"),
		obs("./deploy.sh --env prod --tag v2"),
		obs("./deploy.sh --env dev --tag v3"),
	})
	if a.Key == "" || a.Key != b.Key {
		t.Errorf("key should be stable across values/evolution: %q vs %q", a.Key, b.Key)
	}
	// And the templates legitimately differ (tag became a blank).
	if a.Template == b.Template {
		t.Errorf("expected skeleton to evolve: both %q", a.Template)
	}
}

func TestDeterministicOrdering(t *testing.T) {
	in := []Observation{
		obs("./deploy.sh --env a --tag v1"), obs("./deploy.sh --env b --tag v2"), obs("./deploy.sh --env c --tag v3"),
		obs("terraform apply -var env=a"), obs("terraform apply -var env=b"),
		obs("kubectl -n a logs -f x"), obs("kubectl -n b logs -f x"), obs("kubectl -n c logs -f x"), obs("kubectl -n d logs -f x"),
	}
	first := Detect(in, Options{MinCount: 3, Now: ref})
	for i := 0; i < 5; i++ {
		again := Detect(in, Options{MinCount: 3, Now: ref})
		if len(again) != len(first) {
			t.Fatalf("non-deterministic length")
		}
		for j := range first {
			if again[j].Template != first[j].Template {
				t.Fatalf("non-deterministic order at %d: %q vs %q", j, again[j].Template, first[j].Template)
			}
		}
	}
	// kubectl ran most (4×) → should rank first.
	if len(first) == 0 || !strings.HasPrefix(first[0].Template, "kubectl") {
		t.Errorf("expected kubectl candidate first, got %+v", first)
	}
}
