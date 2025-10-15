package path

import (
	"container/list"
	"lem-in/internal/model"
	"sort"
)

/*
MultiPath (FINAL):
------------------
Return the maximum set of START→END paths that are **room-disjoint** (vertex-disjoint)
except for the Start and End rooms.

Why we use Max-Flow (Edmonds–Karp with node-splitting)
------------------------------------------------------
The earlier greedy "BFS then remove rooms" can pick a path that uses a choke room,
blocking a third (or nth) path that actually exists (your example01 stuck at 2 paths).
Max-flow with node-splitting enforces capacity=1 through each intermediate room and
finds ALL room-disjoint paths.

Important correction for example02:
-----------------------------------
Give each graph edge (u_out → v_in) **capacity = 1**.
If edges are "infinite", a single direct Start—End edge can carry many flow units,
and we reconstruct many identical direct paths, which the scheduler treats as many
distinct paths → all ants finish in one turn. Capacity=1 avoids that by ensuring
edge-disjointness too.
*/

func MultiPath(g *model.Graph, maxPaths int) []*model.Path {
	if g == nil || g.Start == nil || g.End == nil || len(g.Rooms) == 0 {
		return nil
	}

	// ---- Stable name ordering for deterministic results ----
	names := make([]string, 0, len(g.Rooms))
	for name := range g.Rooms {
		names = append(names, name)
	}
	sort.Strings(names)

	// Index each room (stable) -> integer id (0..N-1)
	idOf := make(map[string]int, len(names))
	for i, nm := range names {
		idOf[nm] = i
	}

	N := len(names)

	// Node splitting: each room becomes (v_in, v_out)
	inIdx := func(name string) int { return 2 * idOf[name] }
	outIdx := func(name string) int { return 2*idOf[name] + 1 }

	Sout := outIdx(g.Start.Name) // source = Start_out
	Ein := inIdx(g.End.Name)     // sink   = End_in

	// -------- Residual graph (adjacency with forward+reverse edges) --------
	type edge struct {
		to   int
		rev  int
		cap  int
		flow int
	}
	// Graph size = 2 nodes per room (in+out)
	graph := make([][]edge, 2*N) // total split nodes = 2*N

	addEdge := func(u, v, c int) {
		// forward
		graph[u] = append(graph[u], edge{to: v, rev: len(graph[v]), cap: c, flow: 0})
		// reverse
		graph[v] = append(graph[v], edge{to: u, rev: len(graph[u]) - 1, cap: 0, flow: 0})
	}

	// Capacities:
	// - For each room v, add v_in -> v_out with capacity 1 (room can be used by at most one path).
	// - For Start and End, allow "infinite" capacity so many paths can pass those rooms.
	const INF = 1_000_000
	for _, nm := range names {
		capacity := 1
		if nm == g.Start.Name || nm == g.End.Name {
			capacity = INF
		}
		addEdge(inIdx(nm), outIdx(nm), capacity)
	}

	// For each undirected link u—v, add u_out -> v_in and v_out -> u_in with **capacity 1**.
	// This makes edges themselves non-shareable across distinct paths, preventing duplicate
	// "direct" paths (start->end) and giving a clean, finite set of unique paths.
	for _, nm := range names {
		u := g.Rooms[nm]
		// Sort neighbour names to keep construction deterministic
		nbs := make([]string, 0, len(u.Links))
		for _, nb := range u.Links {
			nbs = append(nbs, nb.Name)
		}
		sort.Strings(nbs)
		for _, vn := range nbs {
			// Add both directions (graph is undirected)
			addEdge(outIdx(nm), inIdx(vn), 1) // <<--- edge capacity is ONE (critical fix)
		}
	}

	// -------- Edmonds–Karp (BFS-based max-flow) from S_out to E_in --------
	source := Sout // start OUT
	sink := Ein    // end IN

	type parentInfo struct {
		u  int // parent node
		ei int // edge index in graph[u]
	}

	var totalFlow int
	bfs := func() ([]parentInfo, int) {
		par := make([]parentInfo, len(graph))
		for i := range par {
			par[i] = parentInfo{-1, -1}
		}
		q := list.New()
		q.PushBack(source)
		par[source] = parentInfo{source, -1}

		for q.Len() > 0 {
			u := q.Remove(q.Front()).(int)
			if u == sink {
				break
			}
			for ei := range graph[u] {
				e := graph[u][ei]
				if par[e.to].u == -1 && e.cap-e.flow > 0 {
					par[e.to] = parentInfo{u, ei}
					q.PushBack(e.to)
				}
			}
		}
		if par[sink].u == -1 {
			return nil, 0
		}

		// Compute bottleneck (here it's 1, but we do it properly).
		bneck := INF
		for v := sink; v != source; {
			pr := par[v]
			e := graph[pr.u][pr.ei]
			if e.cap-e.flow < bneck {
				bneck = e.cap - e.flow
			}
			v = pr.u
		}
		if bneck <= 0 {
			return nil, 0
		}

		// Apply augmentation
		for v := sink; v != source; {
			pr := par[v]
			fe := &graph[pr.u][pr.ei]
			fe.flow += bneck
			// reverse edge
			re := &graph[fe.to][fe.rev]
			re.flow -= bneck
			v = pr.u
		}
		return par, bneck
	}

	// Repeatedly find augmenting paths until no more exist or we hit maxPaths.
	// Each augmentation adds one unit of flow (= one path).
	// We stop if we reach maxPaths, if specified (>0).
	// The main Edmonds-Karp loop:
	for {
		_, pushed := bfs()
		if pushed == 0 {
			break
		}
		totalFlow += pushed
		if maxPaths > 0 && totalFlow >= maxPaths {
			break
		}
	}

	if totalFlow == 0 {
		return nil
	}

	// -------- Reconstruct each path from the final flow --------
	// Each unit of flow gives a room-disjoint (and edge-disjoint) path from S_out to E_in.
	// We greedily trace paths, consuming one unit of flow along used forward edges.
	paths := make([]*model.Path, 0, totalFlow)

	// For stable recon, precompute, for each node, an ordered list of outgoing edge indices
	// sorted by destination's printable "room name" order, for deterministic output.
	roomNameOf := func(idx int) string {
		// idx is either inIdx(name) or outIdx(name); map both to the same name
		return names[idx/2]
	}
	type outEdge struct {
		ei     int
		toName string
	}
	sortedOuts := make([][]outEdge, len(graph))
	for u := range graph {
		outs := make([]outEdge, 0, len(graph[u]))
		for ei, e := range graph[u] {
			outs = append(outs, outEdge{ei: ei, toName: roomNameOf(e.to)})
		}
		sort.Slice(outs, func(i, j int) bool { // stable order by destination name
			if outs[i].toName == outs[j].toName {
				return outs[i].ei < outs[j].ei // secondary sort by edge index
			}
			return outs[i].toName < outs[j].toName // primary sort by room name
		})
		sortedOuts[u] = outs
	}

	consumeFlowAlong := func(u int) (int, bool) {
		// Pick the next forward edge with positive flow out of u, consume 1 unit.
		for _, oe := range sortedOuts[u] {
			ei := oe.ei
			e := &graph[u][ei]
			if e.flow > 0 { // carry one path unit
				e.flow -= 1
				// reverse edge +1
				re := &graph[e.to][e.rev]
				re.flow += 1
				return e.to, true
			}
		}
		return -1, false
	}

	// Trace paths while there is still flow out of the source.
	for {
		// If no positive flow leaves source anymore, we reconstructed all paths.
		nextFromSource, ok := consumeFlowAlong(source)
		if !ok {
			break // No more flow from source = no more paths
		}

		// Start building the readable path (room names). Begin at Start.
		namePath := []string{g.Start.Name}

		cur := nextFromSource
		// cur is expected to be some X_in
		for cur != Ein { // while not at End_in (sink)
			// Step 1: we just entered v_in; record v in the path.
			vName := roomNameOf(cur)
			if vName != g.End.Name { // we’ll append End when we actually reach sink
				namePath = append(namePath, vName)
			}

			// Step 2: consume flow across v_in -> v_out (room capacity edge).
			next, ok2 := consumeFlowAlong(cur)
			if !ok2 {
				// Should not happen in a consistent flow; bail out gracefully.
				break
			}
			cur = next // now at v_out

			// Step 3: consume along an original graph edge v_out -> w_in.
			next, ok3 := consumeFlowAlong(cur)
			if !ok3 {
				break
			}
			cur = next // now at w_in (or sink if w is End)
		}

		// Finally append End and publish the path.
		namePath = append(namePath, g.End.Name)

		// Convert room names to pointers
		roomPath := make([]*model.Room, 0, len(namePath))
		for _, nm := range namePath {
			roomPath = append(roomPath, g.Rooms[nm])
		}
		paths = append(paths, &model.Path{Rooms: roomPath, Length: len(roomPath) - 1})

		if maxPaths > 0 && len(paths) >= maxPaths {
			break
		}
	}

	return paths
}
