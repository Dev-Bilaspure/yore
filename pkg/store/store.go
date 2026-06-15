// Package store is yore's command-memory: an append-only log of the commands
// you actually run, together with the context that makes them useful (the
// directory they ran in, their exit status, and how long they took).
//
// The log is plain JSON Lines (one event per line) so it stays simple,
// dependency-free, and easy to inspect. Aggregation and ranking happen in
// memory in the layers above; this package only owns the on-disk format and IO.
package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ExitUnknown marks an event whose exit status was not recorded (for example,
// commands seeded from a plain history file).
const ExitUnknown = -1

// Event is a single recorded command execution.
type Event struct {
	// Time is when the command was run.
	Time time.Time `json:"time"`
	// Command is the command line that was executed.
	Command string `json:"cmd"`
	// Dir is the working directory the command ran in ("" if unknown).
	Dir string `json:"dir,omitempty"`
	// Exit is the command's exit status, or ExitUnknown.
	Exit int `json:"exit"`
	// Duration is how long the command took (0 if unknown).
	Duration time.Duration `json:"dur,omitempty"`
	// Shell is the shell that produced the event ("zsh", "bash", ...).
	Shell string `json:"shell,omitempty"`
}

// Succeeded reports whether the command is known to have exited successfully.
// An unknown exit status is treated as success so that imported history (which
// has no exit codes) is not penalised.
func (e Event) Succeeded() bool { return e.Exit == 0 || e.Exit == ExitUnknown }

// Store is an append-only event log backed by a single file.
type Store struct {
	path string
}

// New returns a Store backed by the file at path. The file and its parent
// directory are created lazily on the first append.
func New(path string) *Store { return &Store{path: path} }

// Default returns a Store at the user's XDG data location
// ($XDG_DATA_HOME/yore/history.jsonl, falling back to ~/.local/share).
func Default() (*Store, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	return New(filepath.Join(dir, "history.jsonl")), nil
}

// Path returns the backing file path.
func (s *Store) Path() string { return s.path }

// Append writes one event to the log, creating the file if needed. It is safe
// to call concurrently from separate processes: each record is a single line
// written with an append-mode handle, which the OS keeps atomic for small
// writes.
func (s *Store) Append(e Event) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = f.Write(line)
	return err
}

// Load reads every event from the log in chronological (file) order. A missing
// log is not an error: it returns an empty slice. Malformed lines are skipped
// so a single corrupt record never makes the whole history unreadable.
func (s *Store) Load() ([]Event, error) {
	f, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return parse(f)
}

// IsEmpty reports whether the log has no events yet (missing or zero-length).
func (s *Store) IsEmpty() bool {
	info, err := os.Stat(s.path)
	return err != nil || info.Size() == 0
}

func parse(r io.Reader) ([]Event, error) {
	var events []Event
	sc := bufio.NewScanner(r)
	// Commands can be long (heredocs, one-liners); allow generous lines.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip corrupt record
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil {
		return events, err
	}
	return events, nil
}

// dataDir resolves the directory for yore's data, honouring XDG_DATA_HOME.
func dataDir() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "yore"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "yore"), nil
}
