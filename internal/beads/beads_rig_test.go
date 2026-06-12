package beads

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFormatRigDescription(t *testing.T) {
	tests := []struct {
		name    string
		rigName string
		fields  *RigFields
		want    []string
	}{
		{
			name:    "nil fields",
			rigName: "gastown",
			fields:  nil,
			want:    nil, // empty string
		},
		{
			name:    "all fields",
			rigName: "gastown",
			fields: &RigFields{
				Repo:   "git@github.com:user/gastown.git",
				Prefix: "gt",
				State:  RigStateActive,
			},
			want: []string{
				"Rig identity bead for gastown.",
				"repo: git@github.com:user/gastown.git",
				"prefix: gt",
				"state: active",
			},
		},
		{
			name:    "partial fields",
			rigName: "beads",
			fields: &RigFields{
				Prefix: "bd",
			},
			want: []string{
				"Rig identity bead for beads.",
				"prefix: bd",
			},
		},
		{
			name:    "empty fields no repo/prefix/state lines",
			rigName: "empty",
			fields:  &RigFields{},
			want: []string{
				"Rig identity bead for empty.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRigDescription(tt.rigName, tt.fields)
			if tt.want == nil {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
				return
			}
			for _, line := range tt.want {
				if !strings.Contains(got, line) {
					t.Errorf("missing line %q in output:\n%s", line, got)
				}
			}
		})
	}
}

func TestParseRigFields(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want *RigFields
	}{
		{
			name: "empty description",
			desc: "",
			want: &RigFields{},
		},
		{
			name: "full rig description",
			desc: `Rig identity bead for gastown.

repo: git@github.com:user/gastown.git
prefix: gt
state: active`,
			want: &RigFields{
				Repo:   "git@github.com:user/gastown.git",
				Prefix: "gt",
				State:  RigStateActive,
			},
		},
		{
			name: "null values become empty",
			desc: "repo: null\nprefix: bd\nstate: null",
			want: &RigFields{
				Repo:   "",
				Prefix: "bd",
				State:  "",
			},
		},
		{
			name: "only prefix",
			desc: "prefix: bd",
			want: &RigFields{
				Prefix: "bd",
			},
		},
		{
			name: "state maintenance",
			desc: "state: maintenance\nprefix: gt",
			want: &RigFields{
				State:  RigStateMaintenance,
				Prefix: "gt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRigFields(tt.desc)
			if got.Repo != tt.want.Repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.want.Repo)
			}
			if got.Prefix != tt.want.Prefix {
				t.Errorf("Prefix = %q, want %q", got.Prefix, tt.want.Prefix)
			}
			if got.State != tt.want.State {
				t.Errorf("State = %q, want %q", got.State, tt.want.State)
			}
		})
	}
}

func TestRigFieldsRoundTrip(t *testing.T) {
	original := &RigFields{
		Repo:   "git@github.com:user/gastown.git",
		Prefix: "gt",
		State:  RigStateActive,
	}

	formatted := FormatRigDescription("gastown", original)
	parsed := ParseRigFields(formatted)

	if parsed.Repo != original.Repo {
		t.Errorf("Repo: got %q, want %q", parsed.Repo, original.Repo)
	}
	if parsed.Prefix != original.Prefix {
		t.Errorf("Prefix: got %q, want %q", parsed.Prefix, original.Prefix)
	}
	if parsed.State != original.State {
		t.Errorf("State: got %q, want %q", parsed.State, original.State)
	}
}

func TestRigBeadID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"gastown", "gt-rig-gastown"},
		{"beads", "gt-rig-beads"},
		{"my-rig", "gt-rig-my-rig"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RigBeadID(tt.name); got != tt.want {
				t.Errorf("RigBeadID(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRigBeadIDWithPrefix(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		want   string
	}{
		{"gt", "gastown", "gt-rig-gastown"},
		{"bd", "beads", "bd-rig-beads"},
		{"hq", "town", "hq-rig-town"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"-"+tt.name, func(t *testing.T) {
			if got := RigBeadIDWithPrefix(tt.prefix, tt.name); got != tt.want {
				t.Errorf("RigBeadIDWithPrefix(%q, %q) = %q, want %q", tt.prefix, tt.name, got, tt.want)
			}
		})
	}
}

func installMockBDCreateRigRecorder(t *testing.T, logPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test uses Unix shell script mock for bd")
	}

	binDir := t.TempDir()
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$MOCK_BD_LOG"

cmd=""
for arg in "$@"; do
  case "$arg" in
    --*) ;;
    *) cmd="$arg"; break ;;
  esac
done

case "$cmd" in
  config)
    if echo "$*" | grep -q "get types.custom"; then
      echo "agent,role,rig,convoy,slot,queue,event,message,molecule,gate,merge-request"
    fi
    if echo "$*" | grep -q "get types.infra"; then
      echo "agent,role,message"
    fi
    exit 0
    ;;
  create)
    printf '{"id":"gt-rig-gastown","title":"gastown","status":"open","issue_type":"rig","labels":["gt:rig"],"ephemeral":false}\n'
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "bd"), []byte(script), 0755); err != nil {
		t.Fatalf("write mock bd: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MOCK_BD_LOG", logPath)
}

func TestCreateRigBeadUsesDurableRigType(t *testing.T) {
	workDir := t.TempDir()
	beadsDir := filepath.Join(workDir, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(workDir, "bd.log")
	installMockBDCreateRigRecorder(t, logPath)
	ResetEnsuredDirs()

	b := NewWithBeadsDir(workDir, beadsDir)
	issue, err := b.CreateRigBead("gastown", &RigFields{
		Prefix: "gt",
		State:  RigStateActive,
	})
	if err != nil {
		t.Fatalf("CreateRigBead: %v", err)
	}
	if issue.Type != "rig" {
		t.Fatalf("issue.Type = %q, want rig", issue.Type)
	}
	if !HasLabel(issue, "gt:rig") {
		t.Fatalf("created rig bead missing gt:rig label: %+v", issue.Labels)
	}
	if issue.Ephemeral {
		t.Fatal("new rig bead should be durable, got ephemeral=true")
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read mock bd log: %v", err)
	}
	logOutput := string(logData)
	for _, want := range []string{
		"config set types.infra agent,role,message",
		"create --json --id=gt-rig-gastown",
		"--labels=gt:rig",
		"--type=rig",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("mock bd log missing %q:\n%s", want, logOutput)
		}
	}
	if strings.Contains(logOutput, "--type=task") {
		t.Fatalf("CreateRigBead used task type instead of rig type:\n%s", logOutput)
	}
}

func TestValidRigState(t *testing.T) {
	tests := []struct {
		state RigState
		want  bool
	}{
		{RigStateActive, true},
		{RigStateArchived, true},
		{RigStateMaintenance, true},
		{"", false},
		{"invalid", false},
		{"ACTIVE", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := ValidRigState(tt.state); got != tt.want {
				t.Errorf("ValidRigState(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
