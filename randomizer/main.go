package randomizer

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// attempt to write rom data to a file and print summary info.
func writeRom(b []byte, outPath string, sum []byte) error {
	// write file
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return err
	}

	// print summary
	fmt.Println("SHA-1 sum:", hex.EncodeToString(sum))
	fmt.Println("wrote new ROM to '" + outPath + "'")

	return nil
}

// read the specified file into a slice of bytes, returning an error if the
// read fails or if the file is an invalid rom. also returns the game as an
// int.
func readGivenRom(filename string) ([]byte, int, error) {
	// read file
	f, err := os.Open(filename)
	if err != nil {
		return nil, GAME_UNDEFINED, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, GAME_UNDEFINED, err
	}

	// check file data
	if !romIsAges(b) && !romIsSeasons(b) {
		return nil, GAME_UNDEFINED,
			fmt.Errorf("%s is not an oracles ROM", filename)
	}
	if romIsJp(b) {
		return nil, GAME_UNDEFINED,
			fmt.Errorf("%s is a JP ROM; only US is supported", filename)
	}
	if !romIsVanilla(b) {
		return nil, GAME_UNDEFINED,
			fmt.Errorf("%s is an unrecognized oracles ROM", filename)
	}

	game := ternary(romIsSeasons(b), GAME_SEASONS, GAME_AGES).(int)
	return b, game, nil
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
	case GAME_SEASONS:
		romFilename = seasons
	case GAME_AGES:
		romFilename = ages
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
		pause()
		return
	}

	yamlPath := os.Args[1]
	if !strings.HasSuffix(yamlPath, INPUT_FILE_EXTENSION) {
		fmt.Println("Input file must be a " + INPUT_FILE_EXTENSION + " file.")
		pause()
		return
	}

	fmt.Println("Reading input file...")
	ri, err := parseYamlInput(yamlPath)
	if err != nil {
		fatal(err)
		return
	}

	// if rom is to be randomized, infile must be non-empty after switch
	dirName, romFilename := getInputRomPath(ri.game)
	if dirName == "" {
		fmt.Println("No vanilla US ROM found.")
		pause()
		return
	}

	// get input for instance
	b, game, err := readGivenRom(filepath.Join(dirName, romFilename))
	if err != nil {
		fatal(err)
		return
	}

	rom := newRomState(b, game)
	rom.itemSlots = rom.loadSlots()
	rom.initBanks(ri)

	fmt.Println("Patching '" + romFilename + "'...")

	// write roms
	sum, err := rom.setData(ri)
	if err != nil {
		fatal(err)
		return
	}

	var outPath = strings.Replace(yamlPath, INPUT_FILE_EXTENSION, ".gbc", 1)
	if err = writeRom(rom.data, outPath, sum); err != nil {
		fatal(err)
		return
	}

	pause()
}
