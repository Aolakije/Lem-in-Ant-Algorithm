package scheduler

import (
	"fmt"
	"lem-in/internal/model"
	"sort"
	"strings"
)

// Run simulates ants moving along given paths, printing moves each turn in the "Lx-room" format.
//
// KEY RULES of "lem-in":
// 1) Each turn prints a space-separated list of moves "L<antID>-<roomName>".
// 2) A room (except start and end) can contain at most one ant at a time (room exclusivity).
// 3) Multiple ants may leave the start in the same turn (one per chosen path if the first room is free).
// 4) Multiple ants may reach the end in the same turn.
// 5) Edges do NOT need to be locked: the constraint is on rooms, not edges.
// 6) Makespan is minimised with (L-1) balancing: find minimal T with Σ max(0, T - (L_i - 1)) ≥ ants.
func Run(ants int, paths []*model.Path, g *model.Graph) {
	if ants <= 0 || len(paths) == 0 {
		return
	}

	// Sort paths by length ascending (shorter first)
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Length < paths[j].Length
	})

	// Gather lengths (in edges)
	lens := make([]int, len(paths))
	for i, p := range paths {
		lens[i] = p.Length
	}

	// ---- Optimal pre-allocation (L-1 formula) ----
	T := lens[0] - 1
	if T < 0 {
		T = 0
	}
	for {
		sum := 0
		for _, L := range lens {
			base := L - 1
			if T > base {
				sum += T - base
			}
		}
		if sum >= ants {
			break
		}
		T++
	}

	assigned := make([][]int, len(paths)) // IDs per path
	counts := make([]int, len(paths))
	remain := ants
	for i, L := range lens {
		base := L - 1
		take := 0
		if T > base {
			take = T - base
		}
		if take > remain {
			take = remain
		}
		counts[i] = take
		remain -= take
	}
	for i := 0; i < len(paths) && remain > 0; i++ {
		counts[i]++
		remain--
	}

	// Materialise queues 1..ants
	id := 1
	for i := range paths {
		for k := 0; k < counts[i]; k++ {
			assigned[i] = append(assigned[i], id)
			id++
		}
	}

	// ---- Simulation state ----
	occupied := make(map[string]int) // roomName -> antID (non-start/end only)

	startName := g.Start.Name
	endName := g.End.Name

	type AntState struct {
		ID      int
		PathIdx int
		Pos     int
	}
	antsState := make(map[int]*AntState) // antID -> state (only moving ants)

	// Each turn:
	// 1) Move existing ants forward (back-to-front per path)
	// 2) Start new ants (one per path per turn if first room is free)
	// 3) Print moves
	// Repeat until all ants are in the end.

	// Copy assigned queues to mutable wait queues

	waitQueues := make([][]int, len(paths)) // IDs per path, ants waiting to start.
	for i := range paths {
		waitQueues[i] = append([]int{}, assigned[i]...)
	}

	finished := 0

	for finished < ants {
		var moves []string

		// Move existing ants forward (back-to-front per path)
		for pi, p := range paths {
			posToAnt := make(map[int]int)
			for _, a := range antsState {
				if a.PathIdx == pi && a.Pos > 0 {
					posToAnt[a.Pos] = a.ID
				}
			}
			for pos := p.Length - 1; pos >= 0; pos-- {
				antID, ok := posToAnt[pos]
				if !ok {
					continue
				}
				as := antsState[antID]
				curRoom := p.Rooms[as.Pos].Name
				nextRoom := p.Rooms[as.Pos+1].Name

				nextFree := (nextRoom == endName) || (occupied[nextRoom] == 0)
				if nextFree {
					// Move the ant
					if curRoom != startName && curRoom != endName {
						occupied[curRoom] = 0 // Free current room
					}
					if nextRoom != startName && nextRoom != endName {
						occupied[nextRoom] = antID // Occupy next room
					}
					as.Pos++
					moves = append(moves, fmt.Sprintf("L%d-%s", antID, nextRoom))
					if nextRoom == endName {
						finished++
					}
				}
			}
		}

		// Start new ants (one per path per turn if first room is free)
		for pi, p := range paths {
			if len(waitQueues[pi]) == 0 {
				continue
			}
			if len(p.Rooms) == 2 { // direct path start->end
				antID := waitQueues[pi][0]
				waitQueues[pi] = waitQueues[pi][1:]
				moves = append(moves, fmt.Sprintf("L%d-%s", antID, endName))
				finished++
				continue
			}
			first := p.Rooms[1].Name
			if first == endName || occupied[first] == 0 {
				antID := waitQueues[pi][0]
				waitQueues[pi] = waitQueues[pi][1:]
				antsState[antID] = &AntState{ID: antID, PathIdx: pi, Pos: 1}
				if first != startName && first != endName {
					occupied[first] = antID
				}
				moves = append(moves, fmt.Sprintf("L%d-%s", antID, first))
				if first == endName {
					finished++
				}
			}
		}

		if len(moves) > 0 {
			fmt.Println(strings.Join(moves, " "))
		} else {
			break
		}
	}
}
