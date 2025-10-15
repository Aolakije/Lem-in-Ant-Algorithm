package model

type Room struct {
	Name  string
	X     int
	Y     int
	Links []*Room
}

type Graph struct {
	Rooms map[string]*Room
	Start *Room
	End   *Room
}

type Path struct {
	Rooms  []*Room // includes start and end
	Length int     // number of edges
}

func NewGraph() *Graph {
	return &Graph{Rooms: make(map[string]*Room)}
}

func (g *Graph) AddRoom(name string, x, y int) *Room {
	if r, ok := g.Rooms[name]; ok {
		return r
	}
	r := &Room{Name: name, X: x, Y: y}
	g.Rooms[name] = r
	return r
}

func (g *Graph) AddLink(a, b string) bool {
	ra, aok := g.Rooms[a]
	rb, bok := g.Rooms[b]
	if !aok || !bok || a == b {
		return false
	}
	// ensure not duplicate
	for _, l := range ra.Links {
		if l == rb {
			return false
		}
	}
	ra.Links = append(ra.Links, rb)
	rb.Links = append(rb.Links, ra)
	return true
}
