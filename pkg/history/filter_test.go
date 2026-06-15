package history

import "testing"

func TestFilterRawTrimsAndDropsNoise(t *testing.T) {
	in := []Raw{
		{Command: "  git status  "}, // trimmed, kept
		{Command: ""},               // dropped (empty)
		{Command: "q"},              // dropped (single char)
		{Command: "ls"},             // kept
		{Command: "   "},            // dropped (whitespace only)
	}
	got := filterRaw(in, false)
	want := []string{"git status", "ls"}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Command != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i].Command, want[i])
		}
	}
}

func TestRedaction(t *testing.T) {
	secrets := []string{
		"mysql -u root --password hunter2",
		"export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMIabc123",
		"curl -H 'auth: x' https://api.example.com --token abcdef",
		"echo ghp_abcdefghijklmnopqrstuvwxyz0123456789",
		"aws configure set aws_access_key_id AKIAIOSFODNN7EXAMPLE",
		"deploy --api-key=sk_live_abc123",
	}
	for _, s := range secrets {
		if keepCommand(s, true) {
			t.Errorf("expected %q to be redacted", s)
		}
		// Without redaction enabled, the command must be kept.
		if !keepCommand(s, false) {
			t.Errorf("expected %q to be kept when redaction is off", s)
		}
	}

	safe := []string{
		"git push origin main",
		"docker compose up -d",
		"kubectl get pods -A",
		"echo my password is on a sticky note", // prose, not a credential flag
	}
	for _, s := range safe {
		if !keepCommand(s, true) {
			t.Errorf("expected %q to be kept even with redaction on", s)
		}
	}
}
