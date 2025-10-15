# lem‑in — complete, line‑by‑line explanation

> British English throughout. This is a **teaching** document you can use to explain every decision, Rookie‑friendly comments everywhere.

---

## Table of contents

1. [Problem recap: what `lem-in` expects](#problem)
2. [Key rules that drive code design](#rules)
3. [Final extractor: **Max‑flow with node splitting** — line‑by‑line (`multipath.go` final)](#final-multipath)
4. [Final scheduler (rooms-only exclusivity, no edge locks, (L−1) balancing) — line‑by‑line](#final-scheduler)
5. [Maths behind (L−1) balancing, with a mini proof](#balancing)
6. [Complexity and performance notes](#complexity)
7. [Edge cases and how your code handles them](#edge-cases)
8. [A quick oral defence crib: likely questions & crisp answers](#defence)

---

<a id="problem"></a>
## 1) Problem recap: what `lem-in` expects

You’re given a graph (rooms + links), a start room, an end room, and N ants. You must:

- Print the parsed input back (as the audit does) — your main does this.
- Find multiple **room‑disjoint** paths (start and end can be shared).
- Schedule ants along those paths so that:
  - Each intermediate room holds at most one ant at a time.
  - Many ants can leave start in the same turn (one per path if first room is free).
  - Many ants can reach end on the same turn.
- Minimise the **number of turns** (lines of moves printed).

Audits check *turn count*, not the exact order of moves.

---

<a id="rules"></a>
## 2) Key rules that drive code design

- **Room exclusivity** (except start/end). No two ants in the same intermediate room at once.
- **No edge locking in the spec.** Only rooms are exclusive. (Locking edges creates artificial throttling.)
- **Pipelining**: on a path, ants move like a conveyor belt, one room per turn if the next room is free.
- **Optimal pre‑allocation** uses path lengths **L in edges**: sending `x` ants down a path of length `L` takes `(L−1) + x` turns. Balance ants so all used paths finish together.

---

<a id="final-multipath"></a>
## 3) Final `multipath.go` — **max‑flow with node splitting** (line‑by‑line)

> This is the final extractor you’re running now (with edge caps = 1). Comments inline for teaching.

```go
package path // this file belongs to package `path`

import (
	"container/list" // used for BFS queues in max-flow
	"lem-in/model"  // Graph/Room/Path types
	"sort"          // make outputs deterministic by sorting
)

/*
MultiPath (FINAL):
------------------
Return the maximum set of START→END paths that are **room-disjoint** (vertex-disjoint)
except for the Start and End rooms.

Why we use Max-Flow (Edmonds–Karp with node-splitting)
------------------------------------------------------
The earlier greedy method can miss existing paths (example01 needed 3). Node-splitting
with capacity 1 through intermediate rooms guarantees at most one path uses a room.
Max-flow then finds the maximum number of such paths automatically.

Important correction for example02:
-----------------------------------
Give each graph edge (u_out → v_in) capacity = 1. Otherwise a single direct Start—End
edge could carry many flow units and we’d reconstruct many identical direct paths.
*/

func MultiPath(g *model.Graph, maxPaths int) []*model.Path { // entry point
	if g == nil || g.Start == nil || g.End == nil || len(g.Rooms) == 0 { // validate input graph
		return nil
	}

	// ---- Stable name ordering for deterministic results ----
	names := make([]string, 0, len(g.Rooms)) // collect room names
	for name := range g.Rooms {              // iteration order of maps is random, so we sort
		names = append(names, name)
	}
	sort.Strings(names)

	// Index each room -> integer id (0..N-1)
	idOf := make(map[string]int, len(names))
	for i, nm := range names {
		idOf[nm] = i // assign stable index to each room name
	}

	N := len(names) // number of rooms

	// Node splitting helpers: map room name to in-node and out-node indices
	inIdx := func(name string) int { return 2 * idOf[name] }     // v_in
	outIdx := func(name string) int { return 2*idOf[name] + 1 }  // v_out

	Sout := outIdx(g.Start.Name) // source is Start_out (where flow originates)
	Ein  := inIdx(g.End.Name)    // sink is End_in (where flow terminates)

	// -------- Residual graph structure --------
	type edge struct{ to, rev, cap, flow int } // rev = index of reverse edge; cap = capacity; flow = current flow
	graph := make([][]edge, 2*N)               // total split nodes = 2*N (each room contributes in+out)

	addEdge := func(u, v, c int) { // append forward and reverse edges between u and v
		// forward edge u -> v with capacity c
		graph[u] = append(graph[u], edge{to: v, rev: len(graph[v]), cap: c, flow: 0})
		// reverse edge v -> u with capacity 0 initially
		graph[v] = append(graph[v], edge{to: u, rev: len(graph[u]) - 1, cap: 0, flow: 0})
	}

	// Capacities:
	// - v_in -> v_out has capacity 1 for all intermediate rooms (room can be used by at most one path).
	// - Start and End get large capacity (INF) so multiple paths may originate/terminate.
	const INF = 1_000_000 // sufficiently large to act as "infinite" for our inputs
	for _, nm := range names {
		capacity := 1
		if nm == g.Start.Name || nm == g.End.Name { // start/end are not bottlenecks
			capacity = INF
		}
		addEdge(inIdx(nm), outIdx(nm), capacity) // v_in -> v_out
	}

	// For each undirected link u—v in the original graph, add two directed edges in the split graph:
	// u_out -> v_in and v_out -> u_in. **Capacity is 1** (critical fix for example02).
	for _, nm := range names {
		u := g.Rooms[nm]
		// Build a sorted neighbour list to make the residual graph deterministic
		nbs := make([]string, 0, len(u.Links))
		for _, nb := range u.Links { nbs = append(nbs, nb.Name) }
		sort.Strings(nbs)
		for _, vn := range nbs {
			addEdge(outIdx(nm), inIdx(vn), 1) // capacity ONE per edge direction
		}
	}

	// -------- Edmonds–Karp (BFS-based max-flow) from S_out (source) to E_in (sink) --------
	source := Sout
	sink   := Ein

	type parentInfo struct{ u, ei int } // remember BFS tree: we came to node via graph[u][ei]

	var totalFlow int
	bfs := func() ([]parentInfo, int) { // returns parent array and bottleneck (0 if no augmenting path)
		par := make([]parentInfo, len(graph))
		for i := range par { par[i] = parentInfo{-1, -1} }
		q := list.New()
		q.PushBack(source)
		par[source] = parentInfo{source, -1} // mark source as visited with dummy parent

		for q.Len() > 0 {
			u := q.Remove(q.Front()).(int)
			if u == sink { break } // reached sink; we can reconstruct
			for ei := range graph[u] { // explore residual edges
				e := graph[u][ei]
				if par[e.to].u == -1 && e.cap-e.flow > 0 { // unseen and has residual capacity
					par[e.to] = parentInfo{u, ei}
					q.PushBack(e.to)
				}
			}
		}
		if par[sink].u == -1 { return nil, 0 } // no augmenting path

		// Compute bottleneck along the path (will be 1 here, but we do it properly)
		bneck := INF
		for v := sink; v != source; {
			pr := par[v]
			e := graph[pr.u][pr.ei]
			if e.cap-e.flow < bneck { bneck = e.cap - e.flow }
			v = pr.u
		}
		if bneck <= 0 { return nil, 0 }

		// Augment flow along the path
		for v := sink; v != source; {
			pr := par[v]
			fe := &graph[pr.u][pr.ei] // forward edge
			fe.flow += bneck
			re := &graph[fe.to][fe.rev] // reverse edge
			re.flow -= bneck
			v = pr.u
		}
		return par, bneck
	}

	for { // repeatedly augment until no more augmenting paths
		_, pushed := bfs()
		if pushed == 0 { break }
		totalFlow += pushed
		if maxPaths > 0 && totalFlow >= maxPaths { break } // honour user limit
	}

	if totalFlow == 0 { return nil } // no path from start to end

	// -------- Reconstruct each path from the final flow --------
	// Each unit of flow corresponds to one room-disjoint (and edge-disjoint) path.
	paths := make([]*model.Path, 0, totalFlow)

	// For stability, pre-compute sorted lists of outgoing edges per node by destination room name.
	roomNameOf := func(idx int) string { return names[idx/2] } // map split node back to its room name
	type outEdge struct{ ei int; toName string }
	sortedOuts := make([][]outEdge, len(graph))
	for u := range graph {
		outs := make([]outEdge, 0, len(graph[u]))
		for ei, e := range graph[u] {
			outs = append(outs, outEdge{ei: ei, toName: roomNameOf(e.to)})
		}
		sort.Slice(outs, func(i, j int) bool {
			if outs[i].toName == outs[j].toName { return outs[i].ei < outs[j].ei }
			return outs[i].toName < outs[j].toName
		})
		sortedOuts[u] = outs
	}

	consumeFlowAlong := func(u int) (int, bool) { // take one unit of positive flow out of node u
		for _, oe := range sortedOuts[u] {
			ei := oe.ei
			e := &graph[u][ei]
			if e.flow > 0 { // this edge still carries some path units
				e.flow -= 1                     // consume one unit
				re := &graph[e.to][e.rev]       // update reverse edge
				re.flow += 1
				return e.to, true
			}
		}
		return -1, false
	}

	for { // while there is some flow leaving source, we can trace another path
		nextFromSource, ok := consumeFlowAlong(source)
		if !ok { break }

		namePath := []string{g.Start.Name} // start path with Start
		cur := nextFromSource             // expected to be some X_in
		for cur != Ein {                  // until we reach End_in (sink)
			vName := roomNameOf(cur)       // we are at v_in; record v (except we’ll append End later)
			if vName != g.End.Name {
				namePath = append(namePath, vName)
			}
			next, ok2 := consumeFlowAlong(cur) // consume v_in -> v_out
			if !ok2 { break }
			cur = next
			next, ok3 := consumeFlowAlong(cur) // consume v_out -> w_in (original edge)
			if !ok3 { break }
			cur = next
		}

		namePath = append(namePath, g.End.Name) // finish with End

		// Convert to []*model.Room and append to results
		roomPath := make([]*model.Room, 0, len(namePath))
		for _, nm := range namePath { roomPath = append(roomPath, g.Rooms[nm]) }
		paths = append(paths, &model.Path{Rooms: roomPath, Length: len(roomPath) - 1})

		if maxPaths > 0 && len(paths) >= maxPaths { break }
	}

	return paths // maximum set of room-disjoint paths
}
```

**Why this fixes your cases:**
- `example01`: returns **3** vertex‑disjoint shortest paths ⇒ scheduler can finish in ≤ 8 turns.
- `example02`: returns **2** unique paths (one direct, one via 1 and 2), not 20 clones, because edges have cap=1.

---

<a id="final-scheduler"></a>
## 4) Final `scheduler.go` — rooms‑only, no edge locks, (L−1) balancing (line‑by‑line)

> This is your current scheduler after removing `usedEdges` and fixing the direct‑path check.

```go
package scheduler // package name

import (
	"fmt"          // printing moves
	"lem-in/model" // Graph/Path types
	"sort"         // sort paths by length for stable balancing
	"strings"      // join moves per turn
)

// Run: simulate turns; print moves in "L<id>-<room>"; stop when all ants finished.
func Run(ants int, paths []*model.Path, g *model.Graph) {
	if ants <= 0 || len(paths) == 0 { return } // trivial guard

	// Sort paths ascending by length (shortest first)
	sort.Slice(paths, func(i, j int) bool { return paths[i].Length < paths[j].Length })

	// Gather lengths
	lens := make([]int, len(paths))
	for i, p := range paths { lens[i] = p.Length }

	// ---- Optimal pre-allocation (L-1 formula) ----
	T := lens[0] - 1; if T < 0 { T = 0 }
	for { // find minimal T with Σ max(0, T-(L-1)) ≥ ants
		sum := 0
		for _, L := range lens { base := L - 1; if T > base { sum += T - base } }
		if sum >= ants { break }
		T++
	}

	assigned := make([][]int, len(paths)) // IDs per path
	counts   := make([]int, len(paths))
	remain   := ants
	for i, L := range lens { // initial counts = max(0, T - (L-1))
		base := L - 1; take := 0; if T > base { take = T - base }
		if take > remain { take = remain }
		counts[i] = take; remain -= take
	}
	for i := 0; i < len(paths) && remain > 0; i++ { // distribute any leftovers left→right
		counts[i]++; remain--
	}

	// Materialise queues with ant IDs 1..ants
	id := 1
	for i := range paths { for k := 0; k < counts[i]; k++ { assigned[i] = append(assigned[i], id); id++ } }

	// Occupancy and state
	occupied := make(map[string]int) // roomName -> antID for intermediate rooms
	startName, endName := g.Start.Name, g.End.Name
	type AntState struct{ ID, PathIdx, Pos int } // Pos indexes p.Rooms (0=start, ..., L=end)
	antsState := make(map[int]*AntState)

	waitQueues := make([][]int, len(paths)) // pending ants per path
	for i := range paths { waitQueues[i] = append([]int{}, assigned[i]...) }

	finished := 0
	for finished < ants { // each iteration = one turn
		var moves []string

		// 1) Move existing ants forward, back-to-front per path
		for pi, p := range paths {
			posToAnt := make(map[int]int)
			for _, a := range antsState { if a.PathIdx == pi && a.Pos > 0 { posToAnt[a.Pos] = a.ID } }
			for pos := p.Length - 1; pos >= 0; pos-- {
				antID, ok := posToAnt[pos]; if !ok { continue }
				as := antsState[antID]
				cur, next := p.Rooms[as.Pos].Name, p.Rooms[as.Pos+1].Name
				nextFree := (next == endName) || (occupied[next] == 0) // **rooms-only** exclusivity
				if nextFree {
					if cur != startName && cur != endName { occupied[cur] = 0 }   // free current room
					if next != startName && next != endName { occupied[next] = antID } // claim next
					as.Pos++
					moves = append(moves, fmt.Sprintf("L%d-%s", antID, next))
					if next == endName { finished++ }
				}
			}
		}

		// 2) Start new ants (one per path per turn if first room is free)
		for pi, p := range paths {
			if len(waitQueues[pi]) == 0 { continue }
			if len(p.Rooms) == 2 { // direct path start->end
				antID := waitQueues[pi][0]; waitQueues[pi] = waitQueues[pi][1:]
				moves = append(moves, fmt.Sprintf("L%d-%s", antID, endName)); finished++; continue
			}
			first := p.Rooms[1].Name
			if first == endName || occupied[first] == 0 {
				antID := waitQueues[pi][0]; waitQueues[pi] = waitQueues[pi][1:]
				antsState[antID] = &AntState{ID: antID, PathIdx: pi, Pos: 1}
				if first != startName && first != endName { occupied[first] = antID }
				moves = append(moves, fmt.Sprintf("L%d-%s", antID, first))
				if first == endName { finished++ }
			}
		}

		if len(moves) > 0 { fmt.Println(strings.Join(moves, " ")) } else { break }
	}
}
```

---

<a id="balancing"></a>
## 5) Why the (L−1) formula is right (and how to defend it)

- On a path of length **L edges** (L = number of rooms − 1), the **first** ant needs **L−1 turns** to reach end (because it starts at Start and must traverse L rooms, moving one room per turn; the last move lands in End which can hold many ants).
- Each **additional** ant on the same path is released one turn later (due to the first intermediate room being exclusive), so the *k‑th* ant finishes at `L−1 + k`.
- If we place `x` ants on that path, the **completion time** for that path is `(L−1) + x`.
- To **minimise the makespan** across several paths, choose the smallest **T** such that the total number of ants we can place without exceeding `T` on any path is at least `N`:
  
  `sum over paths i of max(0, T − (L_i − 1)) ≥ N`.
  
  Then assign `x_i = max(0, T − (L_i − 1))` ants to path i (plus a few leftover 1’s if the sum overshoots by less than #paths). This equalises `(L_i − 1) + x_i ≈ T` across used paths → minimal makespan.

---

<a id="complexity"></a>
## 6) Complexity and performance

- **Max‑flow (Edmonds–Karp)** runs in `O(V * E^2)` on the split graph. After node‑splitting, `V ≈ 2|rooms|`, and `E ≈ 2|links| + 2|rooms|`. For small educational inputs it’s easily fast enough and, crucially, correct.
- **Scheduler** is `O(T * (A + P*L))` roughly, where `T` is number of turns, `A` ants, `P` paths, `L` average path length. All operations are map lookups and small loops.

---

<a id="edge-cases"></a>
## 7) Edge cases covered

- **No path**: max‑flow yields `totalFlow==0` → return `nil` paths → scheduler prints nothing.
- **Only direct path**: max‑flow yields exactly one path of length 1 → scheduler sends ants one per turn (pipelined start room occupancy still limits releases when there are other paths).
- **Multiple identical edges**: edge capacity 1 per direction prevents duplicates.
- **Disconnected rooms / dead links**: simply unused; BFS in max‑flow ignores edges with `cap-flow == 0`.
- **Determinism**: sorting room names and neighbour lists keeps path extraction and reconstruction stable.

---

<a id="defence"></a>
## 8) Oral defence crib — likely questions & strong answers

1. **Why no edge locking in the scheduler?**
   - The subject constrains *rooms*, not edges. Locking edges artificially throttles legal parallel moves; removing it enables full pipelining and reduces turns.

2. **Why `(L−1)` and not `L` in the balancing formula?**
   - The first ant finishes after traversing `L` edges, which takes `L−1` turns because movement into End happens on the last printed turn. Each additional ant adds 1 turn. Completion time is `(L−1)+x`.

3. **Why node splitting?**
   - To enforce *room* capacity 1. Splitting `v` into `v_in→v_out` with cap 1 formalises “only one path can pass through room `v`”. Start/End get high caps so multiple paths may share them.

4. **Why edge capacity 1?**
   - Without it, one physical edge could carry many flows, creating many identical direct paths (your `example02` 20‑ants‑in‑one‑turn issue). Capacity 1 yields unique, edge‑disjoint paths.

5. **Is order of moves important?**
   - No. The audit checks the **number of turns** (lines). The move order per line may differ as long as rules are respected.

6. **Why do we move ants back‑to‑front per path?**
   - To avoid leapfrogging; nearer ants never jump past farther ants in the same turn.

7. **What ensures determinism?**
   - Sorting room names, neighbour lists, and reconstruction edges ensures stable path sets and output behaviour.

8. **Can we prove minimal turns?**
   - Given a fixed set of path lengths, `(L−1)` balancing is optimal by equalising completion times across used paths. Max‑flow ensures we don’t under‑use available room‑disjoint paths.

---