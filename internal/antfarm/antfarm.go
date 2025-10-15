package antfarm

import (
	"bufio"
	"strings"

	"lem-in/internal/model"
	"lem-in/internal/parser"
	"lem-in/internal/path"
)

// Farm is a wrapper for parser.Result
type Farm = parser.Result

// AntPosition represents a single ant's position at a given turn
type AntPosition struct {
	AntID     int    `json:"antId"`
	Room      string `json:"room"`
	PathIndex int    `json:"pathIndex"`
}

// ParseInput parses raw input into a Farm
func ParseInput(input string) (*Farm, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	return parser.Parse(scanner)
}

// Suurballe returns the set of room-disjoint paths from start to end
func Suurballe(farm *Farm) [][]*model.Path {
	paths := path.MultiPath(farm.Graph, 0) // 0 = no limit
	if len(paths) == 0 {
		return nil
	}

	// Return as a slice of slices with a single path each
	res := make([][]*model.Path, len(paths))
	for i, p := range paths {
		res[i] = []*model.Path{p}
	}
	return res
}

// Schedule simulates ant movements and returns movements per turn
func Schedule(farm *Farm, paths [][]*model.Path) [][]AntPosition {
	if farm.Ants <= 0 || len(paths) == 0 {
		return nil
	}

	var allTurns [][]AntPosition

	type AntState struct {
		ID      int
		PathIdx int
		Pos     int
	}

	numPaths := len(paths)
	ants := farm.Ants

	// Assign ants to paths in round-robin
	assigned := make([][]int, numPaths)
	antID := 1
	for antID <= ants {
		for i := 0; i < numPaths && antID <= ants; i++ {
			assigned[i] = append(assigned[i], antID)
			antID++
		}
	}

	occupied := make(map[string]int)
	startName := farm.Graph.Start.Name
	endName := farm.Graph.End.Name
	antsState := make(map[int]*AntState)

	waitQueues := assigned
	finished := 0

	for finished < ants {
		var turnPositions []AntPosition

		// Move existing ants along paths
		for pi, pSlice := range paths {
			for _, p := range pSlice {
				posToAnt := make(map[int]int)
				for id, st := range antsState {
					if st.PathIdx == pi {
						posToAnt[st.Pos] = id
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
						if curRoom != startName && curRoom != endName {
							occupied[curRoom] = 0
						}
						if nextRoom != startName && nextRoom != endName {
							occupied[nextRoom] = antID
						}
						as.Pos++
						turnPositions = append(turnPositions, AntPosition{
							AntID:     antID,
							Room:      nextRoom,
							PathIndex: pi,
						})
						if nextRoom == endName {
							finished++
						}
					}
				}
			}
		}

		// Start new ants if possible
		for pi, pSlice := range paths {
			for _, p := range pSlice {
				if len(waitQueues[pi]) == 0 {
					continue
				}
				firstRoom := p.Rooms[1].Name
				if firstRoom == endName || occupied[firstRoom] == 0 {
					newAnt := waitQueues[pi][0]
					waitQueues[pi] = waitQueues[pi][1:]
					antsState[newAnt] = &AntState{ID: newAnt, PathIdx: pi, Pos: 1}
					if firstRoom != startName && firstRoom != endName {
						occupied[firstRoom] = newAnt
					}
					turnPositions = append(turnPositions, AntPosition{
						AntID:     newAnt,
						Room:      firstRoom,
						PathIndex: pi,
					})
					if firstRoom == endName {
						finished++
					}
				}
			}
		}

		if len(turnPositions) > 0 {
			allTurns = append(allTurns, turnPositions)
		} else {
			break
		}
	}

	return allTurns
}
