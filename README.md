# lem-in
A highly optimized ant colony path-finding and scheduling solution that efficiently routes ants from start to end through an interconnected room network.

---

## Table of Contents
1. [Overview](#overview)
2. [Concepts](#concepts)
3. [Usage](#usage)
4. [Algorithms](#algorithms)
5. [Project Structure](#structure)
6. [Visualization](#visualization)
7. [Team](#team)

<a id="overview"></a>
## Overview
Lem-in simulates an ant colony navigating through a network of interconnected rooms. The goal is to move all ants from a start room to an end room in the minimum number of turns, subject to the constraint that each intermediate room can hold at most one ant at a time.

The program reads a map description from a file, finds optimal paths, and outputs the movements of ants turn by turn.

---

<a id="concepts"></a>
## Concepts
* **Rooms**: Vertices in the graph, with unique names and coordinates
* **Links**: Edges connecting rooms
* **Ants**: Entities moving through the network, one step per turn
* **Paths**: Routes from start to end that ants can follow
* **Room-Disjoint Paths**: Paths that don't share any intermediate rooms
* **Turn**: One iteration where multiple ants can move simultaneously

<a id="#usage"></a>
## Usage
### Building and Running
```bash
# Build the project
go build -o lem-in

# Run with an input file
./lem-in example01.txt
```
```
# Run with visualizer
go run cmd/server.go



### Input Format
```bash
# Number of ants
3
# Rooms (name x_coord y_coord)
##start
start 0 0
room1 1 0
room2 2 0
##end
end 3 0
# Links
start-room1
room1-room2
room2-end
start-end
```

### Output Format
First, the program echoes the validated input, then prints ant movements:
```bash
3
##start
start 0 0
room1 1 0
room2 2 0
##end
end 3 0
start-room1
room1-room2
room2-end
start-end

L1-end L2-room1 L3-room1
L2-room2 L3-room2
L2-end L3-end
```

Each line represents one turn. The format `L<ant_id>-<room_name>` indicates ant movements.

---

## Algorithms
### Path Finding

We use a Max-Flow with Node Splitting algorithm (Edmonds-Karp) to find the maximum set of room-disjoint paths:

1. Each room is split into an "in" node and "out" node connected by an edge with capacity 1
2. Start and end rooms get large capacity to allow multiple paths to share them
3. Original graph edges connect out-nodes to in-nodes with capacity 1
4. Max-flow algorithm finds the maximum number of paths that don't share intermediate rooms.

This ensures we find the optimal set of paths that can be used simultaneously.

### Ant Scheduling

We use an (L-1) Balancing algorithm to optimally distribute ants among paths:

1. Sort paths by length (shortest first)
2. Find minimal turn count T where `sum(max(0, T - (L_i - 1)))` ≥ number of ants
3. Assign `max(0, T - (L_i - 1))` ants to each path
4. Simulate turns by moving ants forward when next room is free
5. Start new ants along paths when first room becomes available

This ensures all ants reach the end in the minimum number of turns.

## Project Structure

```bash
├── README.md                     # Project overview and instructions
├── badexample00.txt              # Invalid input example for testing
├── badexample01.txt              # Another invalid input example
├── cmd
│   ├── server.go                 # HTTP server for web visualizer
│   └── visualizer
│       └── templates
│           ├── error.html        # Template for showing errors
│           ├── index.html        # Home page input form
│           └── visualize.html    # SVG visualization page
├── docs
│   └── lem_in_full_line_by_line_walkthrough_from_your_original_code_to_the_final_passing_audit.md  # Detailed walkthrough
├── example00.txt                 # Valid input example
├── example01.txt                 # Valid input example
├── example02.txt                 # Valid input example
├── example03.txt                 # Valid input example
├── example04.txt                 # Valid input example
├── example05.txt                 # Valid input example
├── example06.txt                 # Valid input example
├── example07.txt                 # Valid input example
├── go.mod                        # Go module file
├── internal
│   ├── antfarm
│   │   └── antfarm.go            # Main ant farm logic + parsing wrapper
│   ├── model
│   │   ├── model.go              # Core structs: Room, Path, Graph
│   │   └── model_test.go         # Unit tests for model
│   ├── parser
│   │   ├── parser.go             # Input parsing logic
│   │   └── parser_test.go        # Unit tests for parser
│   ├── path
│   │   └── multipath.go          # Pathfinding: MultiPath (max-flow)
│   └── scheduler
│       ├── scheduler.go          # Ant movement simulation
│       └── scheduler_test.go     # Unit tests for scheduler
├── lem-in                        # Compiled executable or main binary
├── main.go                        # Optional CLI entry point
└── test
    └── testdata
        └── valid
            └── sample.txt        # Sample valid input for testing

```

## Visualization
*Coming soon:* A web-based visualizer that will display:

* The graph structure
* Calculated paths
* Ant movement animation
* Turn-by-turn statistics

The visualizer will help better understand how the algorithm works and provide insights into path efficiency.

## Team
This project was developed by a team of three dedicated developers focused on creating an efficient solution to a complex graph traversal and scheduling problem.

For detailed explanations of the algorithms, see our [documentation](docs/lem_in_full_line_by_line_walkthrough_from_your_original_code_to_the_final_passing_audit.md).