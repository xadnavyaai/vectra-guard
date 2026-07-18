package behavioral

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Node is a single category node in the session action graph.
type Node struct {
	Category ActionCategory `json:"category"`
	Count    int            `json:"count"`
}

// Edge is a directed transition from one category to another.
type Edge struct {
	From     ActionCategory `json:"from"`
	To       ActionCategory `json:"to"`
	Count    int            `json:"count"`
	TotalGap time.Duration  `json:"total_gap_ns"`
}

// SessionGraph is the behavioral profile of a single session.
type SessionGraph struct {
	SessionID string     `json:"session_id"`
	Nodes     []Node     `json:"nodes"`
	Edges     []Edge     `json:"edges"`
	Actions   int        `json:"actions"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// BuildGraph constructs a deterministic SessionGraph from a
// timestamp-ordered slice of TimedActions. Same input produces
// byte-identical ToJSON() output across calls.
func BuildGraph(sessionID string, actions []TimedAction) *SessionGraph {
	sorted := make([]TimedAction, len(actions))
	copy(sorted, actions)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	nodeCounts := make(map[ActionCategory]int)
	type edgeKey struct {
		from, to ActionCategory
	}
	edgeCounts := make(map[edgeKey]int)
	edgeGaps := make(map[edgeKey]time.Duration)

	for i, a := range sorted {
		nodeCounts[a.Category]++
		if i == 0 {
			continue
		}
		prev := sorted[i-1]
		k := edgeKey{from: prev.Category, to: a.Category}
		edgeCounts[k]++
		edgeGaps[k] += a.Timestamp.Sub(prev.Timestamp)
	}

	// Emit nodes in the canonical category order defined by
	// AllCategories() so serialization is stable regardless of map
	// iteration order.
	var nodes []Node
	for _, cat := range AllCategories() {
		if c, ok := nodeCounts[cat]; ok {
			nodes = append(nodes, Node{Category: cat, Count: c})
		}
	}

	// Emit edges sorted lexicographically by (from, to). This must be
	// deterministic for the "same input → byte-identical ToJSON" test.
	var edges []Edge
	for k, c := range edgeCounts {
		edges = append(edges, Edge{
			From:     k.from,
			To:       k.to,
			Count:    c,
			TotalGap: edgeGaps[k],
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	g := &SessionGraph{
		SessionID: sessionID,
		Nodes:     nodes,
		Edges:     edges,
		Actions:   len(sorted),
	}
	if len(sorted) > 0 {
		start := sorted[0].Timestamp
		end := sorted[len(sorted)-1].Timestamp
		g.StartTime = &start
		g.EndTime = &end
	}
	return g
}

// ToJSON serializes the graph as indented JSON. Output is deterministic
// for a given input.
func (g *SessionGraph) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

// NodeDiff describes a single node change between two session graphs.
// A node may be added, removed, or have its count changed.
type NodeDiff struct {
	Category ActionCategory `json:"category"`
	OldCount int            `json:"old_count"`
	NewCount int            `json:"new_count"`
	Delta    int            `json:"delta"` // NewCount - OldCount
}

// EdgeDiff describes a single edge change between two session graphs.
// An edge may be added, removed, or have its count changed.
type EdgeDiff struct {
	From     ActionCategory `json:"from"`
	To       ActionCategory `json:"to"`
	OldCount int            `json:"old_count"`
	NewCount int            `json:"new_count"`
	Delta    int            `json:"delta"` // NewCount - OldCount
}

// GraphDiff is the structural difference between two session graphs.
// Added entries have OldCount=0. Removed entries have NewCount=0.
// Changed entries have both non-zero with a non-zero Delta.
type GraphDiff struct {
	BaseSessionID   string     `json:"base_session_id"`
	TargetSessionID string     `json:"target_session_id"`
	AddedNodes      []NodeDiff `json:"added_nodes"`
	RemovedNodes    []NodeDiff `json:"removed_nodes"`
	ChangedNodes    []NodeDiff `json:"changed_nodes"`
	AddedEdges      []EdgeDiff `json:"added_edges"`
	RemovedEdges    []EdgeDiff `json:"removed_edges"`
	ChangedEdges    []EdgeDiff `json:"changed_edges"`
}

// IsEmpty reports whether two graphs were structurally identical.
func (d *GraphDiff) IsEmpty() bool {
	return len(d.AddedNodes) == 0 && len(d.RemovedNodes) == 0 && len(d.ChangedNodes) == 0 &&
		len(d.AddedEdges) == 0 && len(d.RemovedEdges) == 0 && len(d.ChangedEdges) == 0
}

// DiffGraphs compares two session graphs and returns the structural
// difference. The diff is deterministic for a given input pair: added,
// removed, and changed entries are all emitted in the same canonical
// order BuildGraph uses for nodes and edges.
func DiffGraphs(base, target *SessionGraph) *GraphDiff {
	d := &GraphDiff{}
	if base != nil {
		d.BaseSessionID = base.SessionID
	}
	if target != nil {
		d.TargetSessionID = target.SessionID
	}

	baseNodes := map[ActionCategory]int{}
	targetNodes := map[ActionCategory]int{}
	if base != nil {
		for _, n := range base.Nodes {
			baseNodes[n.Category] = n.Count
		}
	}
	if target != nil {
		for _, n := range target.Nodes {
			targetNodes[n.Category] = n.Count
		}
	}

	for _, cat := range AllCategories() {
		oldC, oldOK := baseNodes[cat]
		newC, newOK := targetNodes[cat]
		switch {
		case !oldOK && newOK:
			d.AddedNodes = append(d.AddedNodes, NodeDiff{Category: cat, OldCount: 0, NewCount: newC, Delta: newC})
		case oldOK && !newOK:
			d.RemovedNodes = append(d.RemovedNodes, NodeDiff{Category: cat, OldCount: oldC, NewCount: 0, Delta: -oldC})
		case oldOK && newOK && oldC != newC:
			d.ChangedNodes = append(d.ChangedNodes, NodeDiff{Category: cat, OldCount: oldC, NewCount: newC, Delta: newC - oldC})
		}
	}

	type edgeKey struct {
		from, to ActionCategory
	}
	baseEdges := map[edgeKey]int{}
	targetEdges := map[edgeKey]int{}
	if base != nil {
		for _, e := range base.Edges {
			baseEdges[edgeKey{e.From, e.To}] = e.Count
		}
	}
	if target != nil {
		for _, e := range target.Edges {
			targetEdges[edgeKey{e.From, e.To}] = e.Count
		}
	}

	// Union of edge keys, then emit sorted by (from, to) using the
	// same canonical category order to keep output deterministic.
	seen := map[edgeKey]bool{}
	var allKeys []edgeKey
	for k := range baseEdges {
		if !seen[k] {
			allKeys = append(allKeys, k)
			seen[k] = true
		}
	}
	for k := range targetEdges {
		if !seen[k] {
			allKeys = append(allKeys, k)
			seen[k] = true
		}
	}
	sort.Slice(allKeys, func(i, j int) bool {
		if allKeys[i].from != allKeys[j].from {
			return allKeys[i].from < allKeys[j].from
		}
		return allKeys[i].to < allKeys[j].to
	})
	for _, k := range allKeys {
		oldC, oldOK := baseEdges[k]
		newC, newOK := targetEdges[k]
		switch {
		case !oldOK && newOK:
			d.AddedEdges = append(d.AddedEdges, EdgeDiff{From: k.from, To: k.to, OldCount: 0, NewCount: newC, Delta: newC})
		case oldOK && !newOK:
			d.RemovedEdges = append(d.RemovedEdges, EdgeDiff{From: k.from, To: k.to, OldCount: oldC, NewCount: 0, Delta: -oldC})
		case oldOK && newOK && oldC != newC:
			d.ChangedEdges = append(d.ChangedEdges, EdgeDiff{From: k.from, To: k.to, OldCount: oldC, NewCount: newC, Delta: newC - oldC})
		}
	}

	return d
}

// ToJSON serializes the diff as indented JSON. Deterministic for a
// given input.
func (d *GraphDiff) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// ToText renders the diff as a human-readable summary suitable for a
// terminal. Empty sections are omitted.
func (d *GraphDiff) ToText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Behavioral diff: %s → %s\n", d.BaseSessionID, d.TargetSessionID)
	if d.IsEmpty() {
		b.WriteString("(no structural differences)\n")
		return b.String()
	}
	if len(d.AddedNodes) > 0 {
		b.WriteString("\nAdded nodes:\n")
		for _, n := range d.AddedNodes {
			fmt.Fprintf(&b, "  + %s  (count %d)\n", n.Category, n.NewCount)
		}
	}
	if len(d.RemovedNodes) > 0 {
		b.WriteString("\nRemoved nodes:\n")
		for _, n := range d.RemovedNodes {
			fmt.Fprintf(&b, "  - %s  (was %d)\n", n.Category, n.OldCount)
		}
	}
	if len(d.ChangedNodes) > 0 {
		b.WriteString("\nChanged nodes:\n")
		for _, n := range d.ChangedNodes {
			sign := "+"
			if n.Delta < 0 {
				sign = ""
			}
			fmt.Fprintf(&b, "  ~ %s  %d → %d  (%s%d)\n", n.Category, n.OldCount, n.NewCount, sign, n.Delta)
		}
	}
	if len(d.AddedEdges) > 0 {
		b.WriteString("\nAdded edges:\n")
		for _, e := range d.AddedEdges {
			fmt.Fprintf(&b, "  + %s → %s  (count %d)\n", e.From, e.To, e.NewCount)
		}
	}
	if len(d.RemovedEdges) > 0 {
		b.WriteString("\nRemoved edges:\n")
		for _, e := range d.RemovedEdges {
			fmt.Fprintf(&b, "  - %s → %s  (was %d)\n", e.From, e.To, e.OldCount)
		}
	}
	if len(d.ChangedEdges) > 0 {
		b.WriteString("\nChanged edges:\n")
		for _, e := range d.ChangedEdges {
			sign := "+"
			if e.Delta < 0 {
				sign = ""
			}
			fmt.Fprintf(&b, "  ~ %s → %s  %d → %d  (%s%d)\n", e.From, e.To, e.OldCount, e.NewCount, sign, e.Delta)
		}
	}
	return b.String()
}

// ToDOT serializes the graph as a Graphviz DOT digraph. Output is
// deterministic for a given input (nodes and edges are emitted in the
// canonical order established by BuildGraph). Render with:
//
//	vg behavioral profile --session <id> --format dot | dot -Tpng -o graph.png
func (g *SessionGraph) ToDOT() string {
	var b strings.Builder

	name := g.SessionID
	if name == "" {
		name = "session"
	}
	fmt.Fprintf(&b, "digraph %q {\n", name)
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=\"rounded,filled\", fillcolor=\"#f5f5f5\", fontname=\"Helvetica\"];\n")
	b.WriteString("  edge [fontname=\"Helvetica\", fontsize=10];\n")
	b.WriteString("\n")

	for _, n := range g.Nodes {
		fmt.Fprintf(&b, "  %q [label=\"%s\\n(%d)\"];\n", string(n.Category), string(n.Category), n.Count)
	}
	if len(g.Nodes) > 0 && len(g.Edges) > 0 {
		b.WriteString("\n")
	}

	for _, e := range g.Edges {
		// Average gap makes the edge labels human-readable. For a
		// single-occurrence edge this is just the gap itself.
		avg := e.TotalGap
		if e.Count > 1 {
			avg = e.TotalGap / time.Duration(e.Count)
		}
		fmt.Fprintf(&b, "  %q -> %q [label=\"%d  (avg %s)\"];\n",
			string(e.From), string(e.To), e.Count, avg)
	}

	b.WriteString("}\n")
	return b.String()
}
