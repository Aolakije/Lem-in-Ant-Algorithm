package parser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"lem-in/internal/model"
)

var (
	errInvalid = errors.New("ERROR: invalid data format")
	roomLineRe = regexp.MustCompile(`^([^\s#L][^\s]*)\s+(-?\d+)\s+(-?\d+)$`)
	linkLineRe = regexp.MustCompile(`^([^\s#L][^\s]*)-([^\s#L][^\s]*)$`)
)

type Result struct {
	Ants          int
	Graph         *model.Graph
	OriginalLines []string // sanitized lines to echo before moves
}

func ParseFile(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(bufio.NewScanner(f))
}

func Parse(scanner *bufio.Scanner) (*Result, error) {
	res := &Result{Graph: model.NewGraph()}
	lines := []string{}
	phase := "ants" // ants -> rooms -> links
	var pendingCommand string
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" {
			return nil, fmt.Errorf("%w, empty line", errInvalid)
		}
		if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "##") { // command
				cmd := line
				if cmd == "##start" || cmd == "##end" {
					pendingCommand = cmd
					lines = append(lines, line)
				} else {
					// ignore unknown command (do not store?) spec says ignore; we will echo anyway
					lines = append(lines, line)
				}
			}
			continue
		}
		// ants count
		if phase == "ants" {
			ants, err := strconv.Atoi(line)
			if err != nil || ants <= 0 {
				return nil, fmt.Errorf("%w", errInvalid)
			}
			res.Ants = ants
			lines = append(lines, line) // allow empty? treat as error for simplicity
			phase = "rooms"
			continue
		}
		if roomLineRe.MatchString(line) && (phase == "rooms") {
			m := roomLineRe.FindStringSubmatch(line)
			name := m[1]
			x, _ := strconv.Atoi(m[2])
			y, _ := strconv.Atoi(m[3])
			if _, exists := res.Graph.Rooms[name]; exists {
				return nil, fmt.Errorf("%w, duplicate room", errInvalid)
			}
			r := res.Graph.AddRoom(name, x, y)
			if pendingCommand == "##start" {
				if res.Graph.Start != nil {
					return nil, fmt.Errorf("%w, multiple start", errInvalid)
				}
				res.Graph.Start = r
			} else if pendingCommand == "##end" {
				if res.Graph.End != nil {
					return nil, fmt.Errorf("%w, multiple end", errInvalid)
				}
				res.Graph.End = r
			}
			pendingCommand = ""
			lines = append(lines, line)
			continue
		}
		// link lines transition phase
		if linkLineRe.MatchString(line) {
			if res.Graph.Start == nil || res.Graph.End == nil {
				return nil, fmt.Errorf("%w, missing start or end", errInvalid)
			}
			phase = "links"
			m := linkLineRe.FindStringSubmatch(line)
			if !res.Graph.AddLink(m[1], m[2]) {
				return nil, fmt.Errorf("%w, invalid link", errInvalid)
			}
			lines = append(lines, line)
			continue
		}
		if phase == "links" { // after first link every next must be link
			return nil, fmt.Errorf("%w, expected link", errInvalid)
		}
		return nil, fmt.Errorf("%w, unrecognized line", errInvalid)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if res.Ants == 0 || res.Graph.Start == nil || res.Graph.End == nil {
		return nil, fmt.Errorf("%w, missing essential data", errInvalid)
	}
	res.OriginalLines = lines
	return res, nil
}
