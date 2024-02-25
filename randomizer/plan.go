package randomizer

import (
    "container/list"
    "fmt"
    "os"
    "strings"
    "strconv"
    "gopkg.in/yaml.v2"
)

type inputData struct {
    items                map[string]string
    dungeons             map[string]string
    portals              map[string]string
    seasons              map[string]string
    hints                map[string]string
    settings             map[string]string
//  oldManRupeeValues    map[string]int
}

func newInputData() *inputData {
    return &inputData{
        items:                make(map[string]string),
        dungeons:             make(map[string]string),
        portals:              make(map[string]string),
        seasons:              make(map[string]string),
        hints:                make(map[string]string),
        settings:             make(map[string]string),
//      oldManRupeeValues:    make(map[string]int),
    }
}

func parseYamlInput(path string) (*inputData, error) {
    fmt.Println("Processing input file " + path)
    yamlFile, err := os.ReadFile(path)
    if err != nil {
        fmt.Println("Could not read YAML file", err)
    }

    rawData := make(map[string]map[string]string)
    yaml.Unmarshal(yamlFile, rawData)

    data := newInputData()
    section := data.items
    for name, contents := range rawData {
        switch(name) {
            case "locations":            section = data.items
            case "dungeon entrances":    section = data.dungeons
            case "subrosia portals":     section = data.portals
            case "default seasons":      section = data.seasons
            case "hints":                section = data.hints
            case "settings":             section = data.settings
            default: return nil, fmt.Errorf("unknown section: %q", name)
        }

        for k,v := range contents {
            section[k] = v
        }
    }

    return data, nil
}


func processLostWoodsItemSequence(sequence string, rom *romState) {
    builder := new(strings.Builder)
    lostWoodsItemSequence := strings.Split(sequence, " ")
    for i:=0 ; i<4 ; i++ {
        seasonByte := 0
        seasonStr := ""
        switch lostWoodsItemSequence[2*i] {
        case "spring": 
            seasonByte = 0
            seasonStr = "\x02\xde"
        case "summer": 
            seasonByte = 1
            seasonStr = "S\x04\xbc"
        case "autumn": 
            seasonByte = 2
            seasonStr = "A\x05\x25"
        case "winter": 
            seasonByte = 3
            seasonStr = "\x03\x7e"
        }

        directionByte := 0
        directionStr := ""
        switch lostWoodsItemSequence[2*i+1] {
        case "up": 
            directionByte = 0
            directionStr = "\x03\x01"
        case "right": 
            directionByte = 1
            directionStr = " \x04\x31"
        case "down": 
            directionByte = 2
            directionStr = " south"
        case "left": 
            directionByte = 3
            directionStr = " \x05\x1e"
        }

        builder.WriteString(seasonStr + directionStr)
        if(i != 3) {
            builder.WriteString("\x01")
        } else { 
            builder.WriteString("\x00")
        }
        mutableName := "lostWoodsItemSequence" + strconv.Itoa(i+1)
        rom.codeMutables[mutableName].new = []byte{byte(seasonByte), byte(directionByte)}
    }
    rom.codeMutables["lostWoodsPhonographText"].new = []byte(builder.String())
}

// like findRoute, but uses a specified configuration instead of a random one.
func makePlannedRoute(rom *romState, data *inputData) (*routeInfo, error) {
    ri := &routeInfo{
        entrances: make(map[string]string),
        usedItems: list.New(),
        usedSlots: list.New(),
    }

    switch data.settings["companion"] {
    case "Ricky":   ri.companion = COMPANION_RICKY
    case "Dimitri": ri.companion = COMPANION_DIMITRI
    case "Moosh":   ri.companion = COMPANION_MOOSH
    default:        ri.companion = sora(rom.game, COMPANION_MOOSH, COMPANION_DIMITRI).(int)
    }
    
    // item slots
    for slot, item := range data.items {
        if !itemFitsInSlot(item, slot) {
            return nil, fmt.Errorf("%s doesn't fit in %s", item, slot)
        }

        ri.usedItems.PushBack(item)
        ri.usedSlots.PushBack(slot)
    }

    // seasons
    if rom.game == gameSeasons {
        // Set Maku Seed to be given at the specified amount of essences
        if str, ok := data.settings["required_essences"]; ok {
            requiredEssences, err := strconv.Atoi(str)
            if err == nil {
                giveMakuTreeScriptAddr := rom.codeMutables["makuStageEssence8"].new
                for i:=7 ; i >= requiredEssences; i-- {
                    mutableName := "makuStageEssence" + strconv.Itoa(i)
                    rom.codeMutables[mutableName].new = giveMakuTreeScriptAddr
                }
            }
        }

        // Set Fool's Ore damage if specified
        if str, ok := data.settings["fools_ore_damage"]; ok {
            foolsOreDamage, err := strconv.Atoi(str); 
            if err == nil {
                foolsOreDamage *= -1
                rom.codeMutables["foolsOreDamage"].new = []byte{byte(foolsOreDamage)}
            }
        }

        if str, ok := data.settings["slot_name"]; ok {
            ri.archipelagoSlotName = str
        }

        // Set heart beep interval if specified
        if str, ok := data.settings["heart_beep_interval"]; ok {
            mutable := rom.codeMutables["heartBeepInterval"]
            switch str {
            case "half":       mutable.new = []byte{0x3f * 2}
            case "quarter":      mutable.new = []byte{0x3f * 4}
            case "disabled":  mutable.new = []byte{0x00, 0xc9}
                                    // c9 => Unconditional return
            }
        }
        
        if str, ok := data.settings["lost_woods_item_sequence"]; ok {
            processLostWoodsItemSequence(str, rom)
        }

        ri.seasons = make(map[string]byte, len(data.seasons))
        for area, season := range data.seasons {
            id := getStringIndex(seasonsById, season)
            if id == -1 {
                return nil, fmt.Errorf("invalid default season: %s", season)
            }
            if getStringIndex(seasonAreas, area) == -1 {
                return nil, fmt.Errorf("invalid season area: %s", area)
            }
            ri.seasons[area] = byte(id)
        }
    } else if len(data.seasons) != 0 {
        return nil, fmt.Errorf("ages doesn't have default seasons")
    }

    // dungeon entrances
    for entrance, dungeon := range data.dungeons {
        entrance = strings.Replace(entrance, " entrance", "", 1)
        for _, s := range []string{entrance, dungeon} {
            if s == "d0" || getStringIndex(dungeonNames[rom.game], s) == -1 {
                return nil, fmt.Errorf("no such dungeon: %s", s)
            }
        }
        ri.entrances[entrance] = dungeon
    }

    // portals
    if rom.game == gameSeasons {
        ri.portals = make(map[string]string, len(data.portals))
        for portal, connect := range data.portals {
            if _, ok := subrosianPortalNames[portal]; !ok {
                return nil, fmt.Errorf("invalid holodrum portal: %s", portal)
            }
            if _, ok := reverseLookup(subrosianPortalNames, connect); !ok {
                return nil, fmt.Errorf("invalid subrosia portal: %s", connect)
            }
            ri.portals[portal] = connect
        }
    } else if len(data.portals) != 0 {
        return nil, fmt.Errorf("ages doesn't have subrosia portals")
    }

    return ri, nil
}

// overwrites regular owl hints with planned ones.
/*
func planOwlHints(p *plan, h *hinter, owlHints map[string]string) error {
    // sanity check first
    for owl, hint := range data.hints {
        hint = strings.Trim(hint, `"`)
        if _, ok := owlHints[owl]; !ok {
            return fmt.Errorf("no such owl: %s", owl)
        }
        if !isValidGameText(hint) {
            return fmt.Errorf("invalid hint text: %q", hint)
        }
    }

    // use hint if planned hint found, placeholder if not
    for owl := range owlHints {
        if hint, ok := data.hints[owl]; ok {
            owlHints[owl] = h.format(strings.Trim(hint, `"`))
        } else {
            owlHints[owl] = "..."
        }
    }

    return nil
}
*/

// returns the index of s in a, or -1 if not found.
func getStringIndex(a []string, s string) int {
    for i, v := range a {
        if v == s {
            return i
        }
    }
    return -1
}
