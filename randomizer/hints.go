package randomizer

/*

import (
	"fmt"
	"strings"
)

// returns a map of owl names to text indexes for the given game.
func getOwlIds(game int) map[string]byte {
	owls := make(map[string]map[string]byte)
	if err := ReadEmbeddedYaml("romdata/owls.yaml", owls); err != nil {
		panic(err)
	}
	return owls[gameNames[game]]
}

// updates the owl statue text data based on the given hints. does not mutate
// anything.
func (rom *romState) setOwlData(owlHints map[string]string) {
	table := rom.codeMutables["owlTextOffsets"]
	text := rom.codeMutables["owlText"]
	builder := new(strings.Builder)
	addr := text.addr.offset
	owlTextIds := getOwlIds(rom.game)

	for _, owlName := range orderedKeys(owlTextIds) {
		hint := owlHints[owlName]
		textId := owlTextIds[owlName]
		str := "\x0c\x00" + strings.ReplaceAll(hint, "\n", "\x01") + "\x00"
		table.new[textId*2] = byte(addr)
		table.new[textId*2+1] = byte(addr >> 8)
		addr += uint16(len(str))
		builder.WriteString(str)
	}

	text.new = []byte(builder.String())

	rom.codeMutables["owlTextOffsets"] = table
	rom.codeMutables["owlText"] = text
}

type hinter struct {
	areas map[string]string
	items map[string]string
}

// returns a new hinter initialized for the given game.
func newHinter(game int) *hinter {
	h := &hinter{
		areas: make(map[string]string),
		items: make(map[string]string),
	}

	// load item names
	itemFiles := []string{
		"hints/common_items.yaml",
		fmt.Sprintf("hints/%s_items.yaml", gameNames[game]),
	}
	for _, filename := range itemFiles {
		if err := ReadEmbeddedYaml(filename, h.items); err != nil {
			panic(err)
		}
	}

	// load area names
	rawAreas := make(map[string][]string)
	areasFilename := fmt.Sprintf("hints/%s_areas.yaml", gameNames[game])
	if err := ReadEmbeddedYaml(areasFilename, rawAreas); err != nil {
		panic(err)
	}

	// transform the areas map from: {final: [internal 1, internal 2]}
	// to: {internal 1: final, internal 2: final}
	for k, a := range rawAreas {
		for _, v := range a {
			h.areas[v] = k
		}
	}

	return h
}

// formats a string for a text box. this doesn't include control characters.
// except for newlines.
func (h *hinter) format(s string) string {
	// split message into words to be wrapped
	words := strings.Split(s, " ")

	// build message line by line
	msg := new(strings.Builder)
	line := ""
	for _, word := range words {
		if len(line) == 0 {
			line += word
		} else if len(line)+len(word) <= 15 {
			line += " " + word
		} else {
			msg.WriteString(line + "\n")
			line = word
		}
	}
	msg.WriteString(line)

	return msg.String()
}

// returns truee iff all the characters in s are in the printable range.
func isValidGameText(s string) bool {
	for _, c := range s {
		if c < ' ' || c > 'z' {
			return false
		}
	}
	return true
}

*/
