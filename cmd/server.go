package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"lem-in/internal/antfarm"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	defaultInput := `9
#rooms
##start
start 0 3
##end
end 10 1
C0 1 0
C1 2 0
C2 3 0
C3 4 0
I4 5 0
I5 6 0
A0 1 2
A1 2 1
A2 4 1
B0 1 4
B1 2 4
E2 6 4
D1 6 3
D2 7 3
D3 8 3
H4 4 2
H3 5 2
F2 6 2
F3 7 2
F4 8 2
G0 1 5
G1 2 5
G2 3 5
G3 4 5
G4 6 5
H3-F2
H3-H4
H4-A2
start-G0
G0-G1
G1-G2
G2-G3
G3-G4
G4-D3
start-A0
A0-A1
A0-D1
A1-A2
A1-B1
A2-end
A2-C3
start-B0
B0-B1
B1-E2
start-C0
C0-C1
C1-C2
C2-C3
C3-I4
D1-D2
D1-F2
D2-E2
D2-D3
D2-F3
D3-end
F2-F3
F3-F4
F4-end
I4-I5
I5-end`

	tmpl := template.Must(template.ParseFiles("cmd/visualizer/templates/index.html"))
	tmpl.Execute(w, map[string]interface{}{
		"DefaultInput": defaultInput,
	})
}

func handleVisualize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	input := r.FormValue("input")

	farm, err := antfarm.ParseInput(input)
	if err != nil {
		renderError(w, input, "Parsing error: "+err.Error())
		return
	}

	if farm.Ants <= 0 {
		renderError(w, input, "Invalid number of ants")
		return
	}
	if farm.Graph.Start == nil {
		renderError(w, input, "Missing start room")
		return
	}
	if farm.Graph.End == nil {
		renderError(w, input, "Missing end room")
		return
	}

	// Get all room-disjoint paths
	paths := antfarm.Suurballe(farm)
	if len(paths) == 0 {
		renderError(w, input, "No valid paths found from start to end")
		return
	}

	// Simulate movements
	movements := antfarm.Schedule(farm, paths)

	// Visualization coordinates
	scale := 50
	offsetX := 100
	offsetY := 100
	height := 600

	roomPositions := make(map[string]map[string]int)
	roomsJSON := []map[string]interface{}{}

	for _, room := range farm.Graph.Rooms {
		x := room.X*scale + offsetX
		y := height - (room.Y*scale + offsetY)

		roomPositions[room.Name] = map[string]int{"x": x, "y": y}

		color := "#060607ff"
		if room == farm.Graph.Start {
			color = "#10b981"
		} else if room == farm.Graph.End {
			color = "#ef4444"
		}

		roomsJSON = append(roomsJSON, map[string]interface{}{
			"name":  room.Name,
			"x":     x,
			"y":     y,
			"color": color,
		})
	}

	tunnelsJSON := []map[string]int{}
	for _, room := range farm.Graph.Rooms {
		for _, link := range room.Links {
			if room.Name < link.Name {
				tunnelsJSON = append(tunnelsJSON, map[string]int{
					"x1": room.X*scale + offsetX,
					"y1": height - (room.Y*scale + offsetY),
					"x2": link.X*scale + offsetX,
					"y2": height - (link.Y*scale + offsetY),
				})
			}
		}
	}

	// Marshal to JSON
	movementsJSON, _ := json.Marshal(movements)
	roomsJSONStr, _ := json.Marshal(roomsJSON)
	tunnelsJSONStr, _ := json.Marshal(tunnelsJSON)
	roomPosJSON, _ := json.Marshal(roomPositions)

	// Prepare path strings for display
	pathStrings := []string{}
	for i, pSlice := range paths {
		for _, p := range pSlice {
			names := []string{}
			for _, r := range p.Rooms {
				names = append(names, r.Name)
			}
			pathStrings = append(pathStrings, fmt.Sprintf("Path %d: %s", i+1, strings.Join(names, " → ")))
		}
	}

	tmpl := template.Must(template.ParseFiles("cmd/visualizer/templates/visualize.html"))
	tmpl.Execute(w, map[string]interface{}{
		"Input":         input,
		"Ants":          farm.Ants,
		"RoomCount":     len(farm.Graph.Rooms),
		"TunnelCount":   len(tunnelsJSON),
		"Movements":     template.JS(movementsJSON),
		"Rooms":         template.JS(roomsJSONStr),
		"Tunnels":       template.JS(tunnelsJSONStr),
		"RoomPositions": template.JS(roomPosJSON),
		"Paths":         pathStrings,
	})
}

func renderError(w http.ResponseWriter, input, errorMsg string) {
	tmpl := template.Must(template.ParseFiles("cmd/visualizer/templates/error.html"))
	tmpl.Execute(w, map[string]interface{}{
		"Input": input,
		"Error": errorMsg,
	})
}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/visualize", handleVisualize)

	fmt.Println("🐜 Lem-in Visualizer Server Starting...")
	fmt.Println("📡 Open your browser to: http://localhost:9090")
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(":9090", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
