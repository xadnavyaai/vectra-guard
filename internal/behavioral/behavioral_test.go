package behavioral

import (
	"bytes"
	"encoding/json"
	"io"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// setTestHome isolates tests from the real user HOME so session.Manager
// never touches real state. Uses t.TempDir() which is auto-cleaned.
func setTestHome(t *testing.T) {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tempHome)
	}
}

// TestCategorize_MapsAllKnownEvents asserts every value of
// session.FileOperation.Operation maps to a non-empty ActionCategory,
// and a representative spread of session.Command binaries do too.
func TestCategorize_MapsAllKnownEvents(t *testing.T) {
	fileOps := []session.FileOperation{
		{Operation: "read", Path: "/tmp/a"},
		{Operation: "create", Path: "/tmp/b"},
		{Operation: "modify", Path: "/tmp/c"},
		{Operation: "delete", Path: "/tmp/d"},
	}
	for _, op := range fileOps {
		cat := CategorizeFileOp(op)
		if cat == "" {
			t.Errorf("CategorizeFileOp(%q) returned empty", op.Operation)
		}
	}

	cmds := []struct {
		cmd  session.Command
		want ActionCategory
	}{
		{session.Command{Command: "git", Args: []string{"status"}}, CatDataRead},
		{session.Command{Command: "git", Args: []string{"commit"}}, CatDataWrite},
		{session.Command{Command: "git", Args: []string{"push"}}, CatNetworkCall},
		{session.Command{Command: "curl", Args: []string{"https://example.com"}}, CatNetworkCall},
		{session.Command{Command: "cat", Args: []string{"README.md"}}, CatDataRead},
		{session.Command{Command: "rm", Args: []string{"-rf", "build"}}, CatDataWrite},
		{session.Command{Command: "npm", Args: []string{"install"}}, CatCodeExec},
		{session.Command{Command: "python3", Args: []string{"script.py"}}, CatCodeExec},
		{session.Command{Command: "ssh", Args: []string{"host"}}, CatAuthAction},
		{session.Command{Command: "gh", Args: []string{"pr", "create"}}, CatExternalAPI},
		{session.Command{Command: "foobar-unknown-tool", Args: nil}, CatInternalCompute},
		{session.Command{Command: "/usr/local/bin/curl", Args: nil}, CatNetworkCall},
	}
	for _, tc := range cmds {
		got := CategorizeCommand(tc.cmd)
		if got == "" {
			t.Errorf("CategorizeCommand(%q) returned empty", tc.cmd.Command)
		}
		if got != tc.want {
			t.Errorf("CategorizeCommand(%q %v) = %q, want %q",
				tc.cmd.Command, tc.cmd.Args, got, tc.want)
		}
	}

	// Exhaustively verify every category appears at least once in the
	// closed set.
	seen := map[ActionCategory]bool{}
	for _, tc := range cmds {
		seen[CategorizeCommand(tc.cmd)] = true
	}
	for _, op := range fileOps {
		seen[CategorizeFileOp(op)] = true
	}
	// We don't require every category to appear — only that no empty
	// string is produced. Above asserts handle the non-empty property.
	_ = seen
}

// TestBuildGraph_CountsEdgesAndNodes uses a fixed 6-action sequence
// and asserts exact node counts, edge counts, and edge weights.
func TestBuildGraph_CountsEdgesAndNodes(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	step := 30 * time.Second
	actions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:README.md"},
		{Timestamp: t0.Add(1 * step), Category: CatFileAccess, Source: "file_op", Label: "read:main.go"},
		{Timestamp: t0.Add(2 * step), Category: CatCodeExec, Source: "command", Label: "go"},
		{Timestamp: t0.Add(3 * step), Category: CatNetworkCall, Source: "command", Label: "curl"},
		{Timestamp: t0.Add(4 * step), Category: CatDataWrite, Source: "file_op", Label: "create:out.txt"},
		{Timestamp: t0.Add(5 * step), Category: CatFileAccess, Source: "file_op", Label: "read:out.txt"},
	}
	g := BuildGraph("session-abc", actions)

	if g.SessionID != "session-abc" {
		t.Errorf("SessionID = %q, want session-abc", g.SessionID)
	}
	if g.Actions != 6 {
		t.Errorf("Actions = %d, want 6", g.Actions)
	}

	nodeByCat := map[ActionCategory]int{}
	for _, n := range g.Nodes {
		nodeByCat[n.Category] = n.Count
	}
	if nodeByCat[CatFileAccess] != 3 {
		t.Errorf("file_access count = %d, want 3", nodeByCat[CatFileAccess])
	}
	if nodeByCat[CatCodeExec] != 1 {
		t.Errorf("code_exec count = %d, want 1", nodeByCat[CatCodeExec])
	}
	if nodeByCat[CatNetworkCall] != 1 {
		t.Errorf("network_call count = %d, want 1", nodeByCat[CatNetworkCall])
	}
	if nodeByCat[CatDataWrite] != 1 {
		t.Errorf("data_write count = %d, want 1", nodeByCat[CatDataWrite])
	}

	// 6 actions → 5 transitions. Expected edges:
	//   file_access→file_access  x1
	//   file_access→code_exec    x1
	//   code_exec→network_call   x1
	//   network_call→data_write  x1
	//   data_write→file_access   x1
	if len(g.Edges) != 5 {
		t.Fatalf("edges = %d, want 5; edges=%+v", len(g.Edges), g.Edges)
	}
	edgeMap := map[string]int{}
	for _, e := range g.Edges {
		edgeMap[string(e.From)+"->"+string(e.To)] = e.Count
	}
	wantEdges := map[string]int{
		"file_access->file_access": 1,
		"file_access->code_exec":   1,
		"code_exec->network_call":  1,
		"network_call->data_write": 1,
		"data_write->file_access":  1,
	}
	for k, v := range wantEdges {
		if edgeMap[k] != v {
			t.Errorf("edge %s = %d, want %d", k, edgeMap[k], v)
		}
	}
}

// TestBuildGraph_IsDeterministic asserts the same input yields
// byte-identical ToJSON output across two independent calls.
func TestBuildGraph_IsDeterministic(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	actions := []TimedAction{
		{Timestamp: t0.Add(5 * time.Second), Category: CatNetworkCall, Source: "command", Label: "curl"},
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:a"},
		{Timestamp: t0.Add(10 * time.Second), Category: CatDataWrite, Source: "file_op", Label: "create:b"},
		{Timestamp: t0.Add(15 * time.Second), Category: CatFileAccess, Source: "file_op", Label: "read:b"},
	}

	g1 := BuildGraph("sess-1", actions)
	g2 := BuildGraph("sess-1", actions)

	j1, err := g1.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 1: %v", err)
	}
	j2, err := g2.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 2: %v", err)
	}
	if !bytes.Equal(j1, j2) {
		t.Errorf("BuildGraph is not deterministic:\nj1=%s\nj2=%s", j1, j2)
	}
}

// TestActionsFromSession_MergesAndOrders verifies the adapter correctly
// merges commands and file ops by timestamp.
func TestActionsFromSession_MergesAndOrders(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	sess := &session.Session{
		ID: "sess-merge",
		Commands: []session.Command{
			{Timestamp: t0.Add(2 * time.Second), Command: "curl", Args: []string{"https://example.com"}},
			{Timestamp: t0.Add(4 * time.Second), Command: "cat", Args: []string{"README.md"}},
		},
		FileOps: []session.FileOperation{
			{Timestamp: t0, Operation: "read", Path: "/tmp/a"},
			{Timestamp: t0.Add(3 * time.Second), Operation: "create", Path: "/tmp/b"},
		},
	}

	actions := actionsFromSession(sess)
	if len(actions) != 4 {
		t.Fatalf("actions = %d, want 4", len(actions))
	}
	// Must be timestamp-ordered.
	for i := 1; i < len(actions); i++ {
		if actions[i].Timestamp.Before(actions[i-1].Timestamp) {
			t.Errorf("out-of-order action at index %d: %v before %v",
				i, actions[i].Timestamp, actions[i-1].Timestamp)
		}
	}
	// First action should be the t0 file op.
	if actions[0].Category != CatFileAccess {
		t.Errorf("actions[0].Category = %q, want file_access", actions[0].Category)
	}
	// Second should be the curl network call.
	if actions[1].Category != CatNetworkCall {
		t.Errorf("actions[1].Category = %q, want network_call", actions[1].Category)
	}
}

// --- Edge cases: empty / nil / single ---

func TestBuildGraph_EmptyActions(t *testing.T) {
	g := BuildGraph("empty", nil)
	if g == nil {
		t.Fatal("BuildGraph(nil) returned nil")
	}
	if g.Actions != 0 {
		t.Errorf("Actions = %d, want 0", g.Actions)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("Nodes = %+v, want empty", g.Nodes)
	}
	if len(g.Edges) != 0 {
		t.Errorf("Edges = %+v, want empty", g.Edges)
	}
	if g.StartTime != nil || g.EndTime != nil {
		t.Errorf("start/end should be nil for empty graph, got start=%v end=%v", g.StartTime, g.EndTime)
	}
	data, err := g.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON empty: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("empty graph JSON not valid: %v", err)
	}
}

func TestBuildGraph_SingleAction(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	actions := []TimedAction{
		{Timestamp: t0, Category: CatCodeExec, Source: "command", Label: "go test"},
	}
	g := BuildGraph("solo", actions)
	if g.Actions != 1 {
		t.Errorf("Actions = %d, want 1", g.Actions)
	}
	if len(g.Nodes) != 1 || g.Nodes[0].Category != CatCodeExec {
		t.Errorf("Nodes = %+v, want single code_exec", g.Nodes)
	}
	if len(g.Edges) != 0 {
		t.Errorf("Edges = %+v, want empty (no transitions for single action)", g.Edges)
	}
	if g.StartTime == nil || g.EndTime == nil {
		t.Fatal("start/end should be set for single-action graph")
	}
	if !g.StartTime.Equal(*g.EndTime) {
		t.Errorf("start != end for single-action: %v vs %v", g.StartTime, g.EndTime)
	}
}

func TestBuildGraph_StartEndTimes(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	actions := []TimedAction{
		// Intentionally out of order to test sorting.
		{Timestamp: t0.Add(10 * time.Second), Category: CatCodeExec, Source: "command", Label: "b"},
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "a"},
		{Timestamp: t0.Add(20 * time.Second), Category: CatDataWrite, Source: "file_op", Label: "c"},
	}
	g := BuildGraph("s", actions)
	if !g.StartTime.Equal(t0) {
		t.Errorf("StartTime = %v, want %v", *g.StartTime, t0)
	}
	if !g.EndTime.Equal(t0.Add(20 * time.Second)) {
		t.Errorf("EndTime = %v, want %v", *g.EndTime, t0.Add(20*time.Second))
	}
}

func TestBuildGraph_EdgeGapAggregation(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	actions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "r1"},
		{Timestamp: t0.Add(5 * time.Second), Category: CatCodeExec, Source: "command", Label: "go"},
		// Second file_access→code_exec transition with a different gap.
		{Timestamp: t0.Add(10 * time.Second), Category: CatFileAccess, Source: "file_op", Label: "r2"},
		{Timestamp: t0.Add(13 * time.Second), Category: CatCodeExec, Source: "command", Label: "go"},
	}
	g := BuildGraph("gap", actions)

	var edge *Edge
	for i := range g.Edges {
		if g.Edges[i].From == CatFileAccess && g.Edges[i].To == CatCodeExec {
			edge = &g.Edges[i]
			break
		}
	}
	if edge == nil {
		t.Fatal("missing file_access->code_exec edge")
	}
	if edge.Count != 2 {
		t.Errorf("edge count = %d, want 2", edge.Count)
	}
	// Gaps: 5s + 3s = 8s total
	if edge.TotalGap != 8*time.Second {
		t.Errorf("total gap = %s, want 8s", edge.TotalGap)
	}
}

func TestBuildGraph_NodesEmittedInCanonicalOrder(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	// Mix categories in arbitrary order.
	actions := []TimedAction{
		{Timestamp: t0, Category: CatNetworkCall, Source: "command", Label: "curl"},
		{Timestamp: t0.Add(1 * time.Second), Category: CatDataRead, Source: "command", Label: "cat"},
		{Timestamp: t0.Add(2 * time.Second), Category: CatAuthAction, Source: "command", Label: "ssh"},
		{Timestamp: t0.Add(3 * time.Second), Category: CatFileAccess, Source: "file_op", Label: "read"},
	}
	g := BuildGraph("canon", actions)

	// Build the expected order by walking AllCategories() and keeping
	// only categories that appear in the input.
	present := map[ActionCategory]bool{
		CatNetworkCall: true, CatDataRead: true,
		CatAuthAction: true, CatFileAccess: true,
	}
	var want []ActionCategory
	for _, c := range AllCategories() {
		if present[c] {
			want = append(want, c)
		}
	}
	if len(g.Nodes) != len(want) {
		t.Fatalf("got %d nodes, want %d", len(g.Nodes), len(want))
	}
	for i, n := range g.Nodes {
		if n.Category != want[i] {
			t.Errorf("node[%d] = %q, want %q", i, n.Category, want[i])
		}
	}
}

func TestBuildGraph_EdgesSortedByFromThenTo(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	actions := []TimedAction{
		{Timestamp: t0, Category: CatNetworkCall, Source: "command", Label: "a"},
		{Timestamp: t0.Add(1 * time.Second), Category: CatDataRead, Source: "command", Label: "b"},
		{Timestamp: t0.Add(2 * time.Second), Category: CatAuthAction, Source: "command", Label: "c"},
		{Timestamp: t0.Add(3 * time.Second), Category: CatCodeExec, Source: "command", Label: "d"},
	}
	g := BuildGraph("sort", actions)
	for i := 1; i < len(g.Edges); i++ {
		prev, cur := g.Edges[i-1], g.Edges[i]
		if prev.From > cur.From || (prev.From == cur.From && prev.To > cur.To) {
			t.Errorf("edges not sorted at index %d: %q->%q then %q->%q",
				i, prev.From, prev.To, cur.From, cur.To)
		}
	}
}

func TestBuildGraph_JSONShape(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	g := BuildGraph("shape", []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "x"},
		{Timestamp: t0.Add(1 * time.Second), Category: CatCodeExec, Source: "command", Label: "y"},
	})
	data, err := g.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{"session_id", "nodes", "edges", "actions", "start_time", "end_time"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing JSON key: %q (output=%s)", key, data)
		}
	}
}

func TestBuildGraph_StableWithIdenticalTimestamps(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	// All four actions share the same timestamp. sort.SliceStable must
	// preserve input order, so the graph must still be deterministic.
	actions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "1"},
		{Timestamp: t0, Category: CatCodeExec, Source: "command", Label: "2"},
		{Timestamp: t0, Category: CatNetworkCall, Source: "command", Label: "3"},
		{Timestamp: t0, Category: CatDataWrite, Source: "file_op", Label: "4"},
	}
	g1 := BuildGraph("stable", actions)
	g2 := BuildGraph("stable", actions)
	j1, _ := g1.ToJSON()
	j2, _ := g2.ToJSON()
	if !bytes.Equal(j1, j2) {
		t.Error("graph not stable across runs with identical timestamps")
	}
}

// --- Taxonomy coverage ---

func TestAllCategories_StableAndComplete(t *testing.T) {
	cats := AllCategories()
	if len(cats) != 8 {
		t.Errorf("expected 8 categories, got %d: %v", len(cats), cats)
	}
	// Every declared constant must appear exactly once.
	want := map[ActionCategory]bool{
		CatDataRead:        true,
		CatDataWrite:       true,
		CatFileAccess:      true,
		CatCodeExec:        true,
		CatExternalAPI:     true,
		CatNetworkCall:     true,
		CatAuthAction:      true,
		CatInternalCompute: true,
	}
	seen := map[ActionCategory]int{}
	for _, c := range cats {
		seen[c]++
		if !want[c] {
			t.Errorf("unknown category in AllCategories: %q", c)
		}
	}
	for c := range want {
		if seen[c] != 1 {
			t.Errorf("category %q appears %d times, want 1", c, seen[c])
		}
	}
	// Order must be identical across calls.
	cats2 := AllCategories()
	for i := range cats {
		if cats[i] != cats2[i] {
			t.Errorf("AllCategories() order not stable at index %d", i)
		}
	}
}

func TestCategorizeFileOp_UnknownFallsBackToFileAccess(t *testing.T) {
	got := CategorizeFileOp(session.FileOperation{Operation: "weird-op"})
	if got != CatFileAccess {
		t.Errorf("unknown file op = %q, want file_access fallback", got)
	}
}

func TestCategorizeFileOp_CaseAndWhitespace(t *testing.T) {
	tests := []struct {
		op   string
		want ActionCategory
	}{
		{"READ", CatFileAccess},
		{"  read  ", CatFileAccess},
		{"Create", CatDataWrite},
		{"DELETE", CatDataWrite},
		{"modify", CatDataWrite},
		{"write", CatDataWrite},
	}
	for _, tc := range tests {
		got := CategorizeFileOp(session.FileOperation{Operation: tc.op})
		if got != tc.want {
			t.Errorf("CategorizeFileOp(%q) = %q, want %q", tc.op, got, tc.want)
		}
	}
}

func TestCategorizeCommand_EmptyBinary(t *testing.T) {
	got := CategorizeCommand(session.Command{Command: ""})
	if got != CatInternalCompute {
		t.Errorf("empty command = %q, want internal_compute", got)
	}
}

func TestCategorizeCommand_GitSubcommands(t *testing.T) {
	tests := []struct {
		args []string
		want ActionCategory
	}{
		{[]string{"status"}, CatDataRead},
		{[]string{"log", "--oneline"}, CatDataRead},
		{[]string{"diff"}, CatDataRead},
		{[]string{"show", "HEAD"}, CatDataRead},
		{[]string{"branch"}, CatDataRead},
		{[]string{"config", "user.name"}, CatDataRead},
		{[]string{"clone", "https://example.com/x.git"}, CatNetworkCall},
		{[]string{"pull"}, CatNetworkCall},
		{[]string{"fetch"}, CatNetworkCall},
		{[]string{"push"}, CatNetworkCall},
		{[]string{"remote", "-v"}, CatNetworkCall},
		{[]string{"commit", "-m", "msg"}, CatDataWrite},
		{[]string{"add", "."}, CatDataWrite},
		{[]string{"rebase"}, CatDataWrite},
		{nil, CatDataWrite}, // git with no args → write bucket
	}
	for _, tc := range tests {
		cmd := session.Command{Command: "git", Args: tc.args}
		got := CategorizeCommand(cmd)
		if got != tc.want {
			t.Errorf("git %v = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestCategorizeCommand_PathStripping(t *testing.T) {
	tests := map[string]ActionCategory{
		"/usr/bin/curl":           CatNetworkCall,
		"/usr/local/bin/python3":  CatCodeExec,
		"/opt/homebrew/bin/gh":    CatExternalAPI,
		`C:\Program Files\ssh`:    CatAuthAction,
		"/bin/cat":                CatDataRead,
		"/usr/bin/rm":             CatDataWrite,
		"/nonstandard/bin/custom": CatInternalCompute,
	}
	for path, want := range tests {
		got := CategorizeCommand(session.Command{Command: path})
		if got != want {
			t.Errorf("CategorizeCommand(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestCategorizeCommand_CaseInsensitive(t *testing.T) {
	got := CategorizeCommand(session.Command{Command: "CURL"})
	if got != CatNetworkCall {
		t.Errorf("CURL = %q, want network_call", got)
	}
}

// TestCategorizeCommand_FullClosedSet enumerates at least one command
// per category and asserts the closed set is covered end-to-end.
func TestCategorizeCommand_FullClosedSet(t *testing.T) {
	cases := []struct {
		cmd  string
		args []string
		want ActionCategory
	}{
		{"cat", nil, CatDataRead},
		{"rm", []string{"-rf", "x"}, CatDataWrite},
		{"curl", []string{"https://x"}, CatNetworkCall},
		{"gh", []string{"pr", "list"}, CatExternalAPI},
		{"ssh", []string{"host"}, CatAuthAction},
		{"python3", []string{"a.py"}, CatCodeExec},
		{"unknown-binary", nil, CatInternalCompute},
	}
	seen := map[ActionCategory]bool{}
	for _, tc := range cases {
		got := CategorizeCommand(session.Command{Command: tc.cmd, Args: tc.args})
		if got != tc.want {
			t.Errorf("%s %v = %q, want %q", tc.cmd, tc.args, got, tc.want)
		}
		seen[got] = true
	}
	// File ops cover the remaining file_access bucket.
	seen[CategorizeFileOp(session.FileOperation{Operation: "read"})] = true

	// We expect every category except maybe a handful to be reachable
	// from this minimal sample. Allow ExternalAPI/AuthAction but
	// require the 6 we listed to all be present.
	required := []ActionCategory{
		CatDataRead, CatDataWrite, CatNetworkCall, CatExternalAPI,
		CatAuthAction, CatCodeExec, CatInternalCompute, CatFileAccess,
	}
	for _, c := range required {
		if !seen[c] {
			t.Errorf("category %q not reached by test cases", c)
		}
	}
}

// --- Adapter tests ---

func TestActionsFromSession_Nil(t *testing.T) {
	got := actionsFromSession(nil)
	if got != nil {
		t.Errorf("actionsFromSession(nil) = %+v, want nil", got)
	}
}

func TestActionsFromSession_EmptySession(t *testing.T) {
	sess := &session.Session{ID: "empty"}
	got := actionsFromSession(sess)
	if len(got) != 0 {
		t.Errorf("empty session → %+v, want empty slice", got)
	}
}

func TestActionsFromSession_PreservesLabelShape(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	sess := &session.Session{
		Commands: []session.Command{
			{Timestamp: t0, Command: "git", Args: []string{"status"}},
		},
		FileOps: []session.FileOperation{
			{Timestamp: t0.Add(1 * time.Second), Operation: "read", Path: "/tmp/a"},
		},
	}
	got := actionsFromSession(sess)
	if len(got) != 2 {
		t.Fatalf("got %d actions, want 2", len(got))
	}
	if got[0].Source != "command" || got[0].Label != "git" {
		t.Errorf("command label wrong: %+v", got[0])
	}
	if got[1].Source != "file_op" || got[1].Label != "read:/tmp/a" {
		t.Errorf("file_op label wrong: %+v", got[1])
	}
}

func TestLoadTimedActions_NilManager(t *testing.T) {
	_, err := LoadTimedActions(nil, "any")
	if err == nil {
		t.Error("expected error for nil manager")
	}
}

func TestLoadTimedActions_MissingSession(t *testing.T) {
	setTestHome(t)
	logger := logging.NewLogger("text", io.Discard)
	mgr, err := session.NewManager(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	_, err = LoadTimedActions(mgr, "does-not-exist")
	if err == nil {
		t.Error("expected error loading nonexistent session")
	}
}

// TestLoadTimedActions_RoundTrip uses a real session.Manager (scoped
// to t.TempDir() / HOME) to round-trip a session through disk and
// verify the adapter pulls it back out correctly.
func TestLoadTimedActions_RoundTrip(t *testing.T) {
	setTestHome(t)
	logger := logging.NewLogger("text", io.Discard)
	mgr, err := session.NewManager(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	sess, err := mgr.Start("test-agent", t.TempDir())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t0 := time.Now()
	if err := mgr.AddCommand(sess, session.Command{
		Timestamp: t0,
		Command:   "curl",
		Args:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}
	if err := mgr.AddFileOperation(sess, session.FileOperation{
		Timestamp: t0.Add(1 * time.Second),
		Operation: "read",
		Path:      "/tmp/x",
		Allowed:   true,
	}); err != nil {
		t.Fatalf("AddFileOperation: %v", err)
	}

	actions, err := LoadTimedActions(mgr, sess.ID)
	if err != nil {
		t.Fatalf("LoadTimedActions: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("actions = %d, want 2: %+v", len(actions), actions)
	}

	// Ordered by timestamp.
	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].Timestamp.Before(actions[j].Timestamp)
	})
	if actions[0].Category != CatNetworkCall {
		t.Errorf("first action category = %q, want network_call", actions[0].Category)
	}
	if actions[1].Category != CatFileAccess {
		t.Errorf("second action category = %q, want file_access", actions[1].Category)
	}

	// Full end-to-end: build a graph from the loaded actions.
	g := BuildGraph(sess.ID, actions)
	if g.Actions != 2 {
		t.Errorf("graph actions = %d, want 2", g.Actions)
	}
}

// TestToDOT_ShapeAndDeterminism asserts the DOT output contains the
// expected digraph header, all node declarations, all edge declarations,
// and is byte-identical across repeated calls for the same graph.
func TestToDOT_ShapeAndDeterminism(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	step := time.Second
	actions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:README.md"},
		{Timestamp: t0.Add(1 * step), Category: CatCodeExec, Source: "command", Label: "go"},
		{Timestamp: t0.Add(2 * step), Category: CatNetworkCall, Source: "command", Label: "curl"},
		{Timestamp: t0.Add(3 * step), Category: CatDataWrite, Source: "file_op", Label: "create:out.txt"},
	}
	g := BuildGraph("session-dot", actions)
	dot1 := g.ToDOT()
	dot2 := g.ToDOT()

	if dot1 != dot2 {
		t.Fatalf("ToDOT not deterministic:\n--- first ---\n%s\n--- second ---\n%s", dot1, dot2)
	}

	wantSubstrings := []string{
		`digraph "session-dot" {`,
		`rankdir=LR;`,
		`"file_access" [label="file_access\n(1)"];`,
		`"code_exec" [label="code_exec\n(1)"];`,
		`"network_call" [label="network_call\n(1)"];`,
		`"data_write" [label="data_write\n(1)"];`,
		`"file_access" -> "code_exec"`,
		`"code_exec" -> "network_call"`,
		`"network_call" -> "data_write"`,
		"}\n",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(dot1, s) {
			t.Errorf("DOT output missing %q\nfull output:\n%s", s, dot1)
		}
	}
}

// TestToDOT_EmptyGraphStillValid ensures an empty graph produces a
// syntactically valid (if trivial) DOT document.
func TestToDOT_EmptyGraphStillValid(t *testing.T) {
	g := BuildGraph("empty-sess", nil)
	dot := g.ToDOT()
	if !strings.Contains(dot, `digraph "empty-sess" {`) {
		t.Errorf("empty graph DOT missing header, got:\n%s", dot)
	}
	if !strings.HasSuffix(dot, "}\n") {
		t.Errorf("empty graph DOT missing closing brace, got:\n%s", dot)
	}
}

// TestDiffGraphs_IdenticalGraphsProduceEmptyDiff asserts that two
// structurally-identical graphs produce a diff with IsEmpty() == true.
func TestDiffGraphs_IdenticalGraphsProduceEmptyDiff(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	step := time.Second
	actions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:a"},
		{Timestamp: t0.Add(step), Category: CatCodeExec, Source: "command", Label: "go"},
		{Timestamp: t0.Add(2 * step), Category: CatDataWrite, Source: "file_op", Label: "create:b"},
	}
	g1 := BuildGraph("base", actions)
	g2 := BuildGraph("target", actions)
	diff := DiffGraphs(g1, g2)

	if !diff.IsEmpty() {
		t.Errorf("expected empty diff for identical graphs, got: %+v", diff)
	}
	if diff.BaseSessionID != "base" || diff.TargetSessionID != "target" {
		t.Errorf("session IDs not recorded: base=%q target=%q", diff.BaseSessionID, diff.TargetSessionID)
	}
}

// TestDiffGraphs_NodeAndEdgeChanges asserts that added, removed, and
// changed nodes and edges are all emitted correctly.
func TestDiffGraphs_NodeAndEdgeChanges(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	step := time.Second

	// Base: file_access → code_exec
	baseActions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:a"},
		{Timestamp: t0.Add(step), Category: CatCodeExec, Source: "command", Label: "go"},
	}
	// Target: file_access → code_exec → network_call (extra action,
	// new node, new edge)
	targetActions := []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:a"},
		{Timestamp: t0.Add(step), Category: CatCodeExec, Source: "command", Label: "go"},
		{Timestamp: t0.Add(2 * step), Category: CatNetworkCall, Source: "command", Label: "curl"},
	}

	baseG := BuildGraph("base", baseActions)
	targetG := BuildGraph("target", targetActions)
	diff := DiffGraphs(baseG, targetG)

	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}

	// Exactly one added node: network_call
	if len(diff.AddedNodes) != 1 || diff.AddedNodes[0].Category != CatNetworkCall || diff.AddedNodes[0].NewCount != 1 {
		t.Errorf("AddedNodes = %+v, want [network_call count=1]", diff.AddedNodes)
	}
	if len(diff.RemovedNodes) != 0 {
		t.Errorf("RemovedNodes = %+v, want []", diff.RemovedNodes)
	}
	// Exactly one added edge: code_exec → network_call
	if len(diff.AddedEdges) != 1 || diff.AddedEdges[0].From != CatCodeExec || diff.AddedEdges[0].To != CatNetworkCall {
		t.Errorf("AddedEdges = %+v, want [code_exec→network_call]", diff.AddedEdges)
	}
	// The file_access → code_exec edge exists in both with count 1,
	// so it should NOT appear in any diff bucket.
	for _, e := range diff.ChangedEdges {
		if e.From == CatFileAccess && e.To == CatCodeExec {
			t.Errorf("file_access→code_exec should not be in ChangedEdges, got %+v", e)
		}
	}
}

// TestDiffGraphs_ChangedNodeCount asserts a node whose count changes
// (but still exists in both) is emitted as ChangedNodes, not Added/Removed.
func TestDiffGraphs_ChangedNodeCount(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	// Base: 1 data_write
	base := BuildGraph("base", []TimedAction{
		{Timestamp: t0, Category: CatDataWrite, Source: "file_op", Label: "create:a"},
	})
	// Target: 3 data_writes
	target := BuildGraph("target", []TimedAction{
		{Timestamp: t0, Category: CatDataWrite, Source: "file_op", Label: "create:a"},
		{Timestamp: t0.Add(time.Second), Category: CatDataWrite, Source: "file_op", Label: "create:b"},
		{Timestamp: t0.Add(2 * time.Second), Category: CatDataWrite, Source: "file_op", Label: "create:c"},
	})
	diff := DiffGraphs(base, target)

	if len(diff.ChangedNodes) != 1 {
		t.Fatalf("ChangedNodes = %+v, want exactly 1", diff.ChangedNodes)
	}
	cn := diff.ChangedNodes[0]
	if cn.Category != CatDataWrite || cn.OldCount != 1 || cn.NewCount != 3 || cn.Delta != 2 {
		t.Errorf("ChangedNode = %+v, want data_write 1→3 delta=+2", cn)
	}
}

// TestDiffGraphs_Deterministic asserts two identical Diff calls produce
// byte-identical JSON output.
func TestDiffGraphs_Deterministic(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	base := BuildGraph("base", []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:a"},
		{Timestamp: t0.Add(time.Second), Category: CatCodeExec, Source: "command", Label: "go"},
	})
	target := BuildGraph("target", []TimedAction{
		{Timestamp: t0, Category: CatFileAccess, Source: "file_op", Label: "read:a"},
		{Timestamp: t0.Add(time.Second), Category: CatNetworkCall, Source: "command", Label: "curl"},
	})

	diff1 := DiffGraphs(base, target)
	diff2 := DiffGraphs(base, target)

	j1, err := diff1.ToJSON()
	if err != nil {
		t.Fatalf("diff1.ToJSON: %v", err)
	}
	j2, err := diff2.ToJSON()
	if err != nil {
		t.Fatalf("diff2.ToJSON: %v", err)
	}
	if !bytes.Equal(j1, j2) {
		t.Errorf("diff JSON not deterministic\nrun1=%s\nrun2=%s", j1, j2)
	}
}

// TestToDOT_EdgeLabelIncludesAverageGap asserts the edge label format
// matches what the blog post renders.
func TestToDOT_EdgeLabelIncludesAverageGap(t *testing.T) {
	t0 := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	actions := []TimedAction{
		// Two data_write→code_exec edges with gaps 1s and 3s → avg 2s.
		{Timestamp: t0, Category: CatDataWrite, Source: "file_op", Label: "write:a"},
		{Timestamp: t0.Add(1 * time.Second), Category: CatCodeExec, Source: "command", Label: "sh"},
		{Timestamp: t0.Add(2 * time.Second), Category: CatDataWrite, Source: "file_op", Label: "write:b"},
		{Timestamp: t0.Add(5 * time.Second), Category: CatCodeExec, Source: "command", Label: "sh"},
	}
	g := BuildGraph("avg-gap", actions)
	dot := g.ToDOT()

	if !strings.Contains(dot, `"data_write" -> "code_exec" [label="2  (avg 2s)"]`) {
		t.Errorf("expected data_write→code_exec edge with avg 2s, got:\n%s", dot)
	}
}
