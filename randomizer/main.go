package randomizer

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
    "encoding/hex"
)

type randomizerOptions struct {
    treewarp bool
    dungeons bool
    portals  bool
    game     int
}

// attempt to write rom data to a file and print summary info.
func writeRom(b []byte, dirName string, filename string, sum []byte) error {
    // write file
    f, err := os.Create(filepath.Join(dirName, filename))
    if err != nil {
        return err
    }
    defer f.Close()
    if _, err := f.Write(b); err != nil {
        return err
    }

    // print summary
    fmt.Println("SHA-1 sum:", hex.EncodeToString(sum))
    fmt.Println("wrote new ROM to '" + filename + "'")

    return nil
}

// read the specified file into a slice of bytes, returning an error if the
// read fails or if the file is an invalid rom. also returns the game as an
// int.
func readGivenRom(filename string) ([]byte, int, error) {
    // read file
    f, err := os.Open(filename)
    if err != nil {
        return nil, gameNil, err
    }
    defer f.Close()
    b, err := ioutil.ReadAll(f)
    if err != nil {
        return nil, gameNil, err
    }

    // check file data
    if !romIsAges(b) && !romIsSeasons(b) {
        return nil, gameNil,
            fmt.Errorf("%s is not an oracles ROM", filename)
    }
    if romIsJp(b) {
        return nil, gameNil,
            fmt.Errorf("%s is a JP ROM; only US is supported", filename)
    }
    if !romIsVanilla(b) {
        return nil, gameNil,
            fmt.Errorf("%s is an unrecognized oracles ROM", filename)
    }

    game := ternary(romIsSeasons(b), gameSeasons, gameAges).(int)
    return b, game, nil
}

// mutates the rom data in-place based on the given route. this doesn't write
// the file.
func setRomData(rom *romState, ri *routeInfo, ropts *randomizerOptions) ([]byte, error) {
    rom.setTreewarp(ropts.treewarp)

    // place selected treasures in slots
    checks := getChecks(ri.usedItems, ri.usedSlots)
    for slot, item := range checks {
        rom.itemSlots[slot].treasure = rom.treasures[item]
    }

    // set season data
    if rom.game == gameSeasons {
        for area, id := range ri.seasons {
            rom.setSeason(inflictCamelCase(area+"Season"), id)
        }
    }

    rom.setAnimal(ri.companion)

    warps := make(map[string]string)
    if ropts.dungeons {
        for k, v := range ri.entrances {
            warps[k] = v
        }
    }
    if ropts.portals {
        for k, v := range ri.portals {
            holodrumV, _ := reverseLookup(subrosianPortalNames, v)
            warps[fmt.Sprintf("%s portal", k)] =
                fmt.Sprintf("%s portal", holodrumV)
        }
    }

    // set slot name
    slotNameAsBytes := []byte(ri.archipelagoSlotName)
    for i := 0 ; i<0x40 ; i++ {
        if i < len(slotNameAsBytes) {
            rom.data[0xFFFC0 + i] = slotNameAsBytes[i]
        } else {
            rom.data[0xFFFC0 + i] = 0x00
        }        
    }

    // do it! (but don't write anything)
    return rom.mutate(warps, ri.archipelagoSlotName, ropts)
}

// search for a vanilla US seasons and ages roms in the executable's directory,
// and return their filenames.
func findVanillaRoms() (dirName, seasons string, ages string, err error) {
    // read slice of file info from executable's dir
    exe, err := os.Executable()
    if err != nil {
        return
    }
    dirName = filepath.Dir(exe)

    dir, err := os.Open(dirName)
    if err != nil {
        return
    }
    defer dir.Close()
    files, err := dir.Readdir(-1)
    if err != nil {
        return
    }

    for _, info := range files {
        // check file metadata
        if info.Size() != 1048576 {
            continue
        }

        // read file
        var f *os.File
        f, err = os.Open(filepath.Join(dirName, info.Name()))
        if err != nil {
            return
        }
        defer f.Close()
        var b []byte
        b, err = ioutil.ReadAll(f)
        if err != nil {
            return
        }

        // check file data
        if !romIsJp(b) && romIsVanilla(b) {
            if romIsAges(b) {
                ages = info.Name()
            } else {
                seasons = info.Name()
            }
        }

        if ages != "" && seasons != "" {
            break
        }
    }

    return
}

// returns the target directory and filenames of input and output files. the
// output filename may be empty, in which case it will be automatically
// determined.
func getInputRomPath(game int) (dir string, filename string) {
    var seasons, ages string
    var err error
    dir, seasons, ages, err = findVanillaRoms()
    if err != nil {
        fatal(err)
        return "", ""
    }

    romFilename := ""
    switch game {
    case gameSeasons: romFilename = seasons
    case gameAges:    romFilename = ages
    }

    if romFilename == "" {
        return "", ""
    }

    return dir, romFilename
}

// the program's entry point.
func Main() {
    fmt.Println()
    fmt.Println("Oracles Archipelago Patcher - " + VERSION)
    fmt.Println("=========================================")
    fmt.Println()

    if len(os.Args) < 2 {
        printErrf("You must supply the path to your input file\n")
    }

    fmt.Println("Reading input file...")
    yamlPath := os.Args[1]
    inputData, err := parseYamlInput(yamlPath)
    if err != nil {
        fatal(err)
        return
    }

    game := 0
    if inputData.settings["game"] == "seasons" {
        game = gameSeasons
    } else if inputData.settings["game"] == "ages" {
        game = gameAges
    }

    // if rom is to be randomized, infile must be non-empty after switch
    dirName, romFilename := getInputRomPath(game)
    if dirName == "" {
        fmt.Println("No vanilla US ROM found.")
        return
    }

    // get input for instance
    b, game, err := readGivenRom(filepath.Join(dirName, romFilename))
    if err != nil {
        fatal(err)
        return
    }
    
    rom := newRomState(b, game)

    // sanity check beforehand
    if errs := rom.verify(); errs != nil {
        fmt.Println(err.Error())
        fatal(errs[0])
        return
    }

    fmt.Println("Patching '" + romFilename + "'...")

    ropts := &randomizerOptions{}
    ropts.treewarp = true
    ropts.game     = game

    route, err := makePlannedRoute(rom, inputData)
    if err != nil {
        fatal(err)
        return
    }
    ropts.dungeons = route.entrances != nil && len(route.entrances) > 0
    ropts.portals = route.portals != nil && len(route.portals) > 0
    
    // accumulate all treasures for reference by log functions
    treasures := make(map[string]*treasure)
    for k, v := range rom.treasures {
        treasures[k] = v
    }

    // write roms
    outfile := strings.Replace(yamlPath, ".yml", ".gbc", 1)

    sum, err := setRomData(rom, route, ropts)
    if err != nil {
        fatal(err)
        return
    }

    if writeRom(rom.data, dirName, outfile, sum); err != nil {
        fatal(err)
        return
    }
}