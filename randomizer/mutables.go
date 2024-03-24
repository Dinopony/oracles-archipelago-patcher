package randomizer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// an instance of ROM data that can be changed by the randomizer.
/*
type mutable interface {
	mutate([]byte) // change ROM bytes
}
*/

// a length of mutable bytes starting at a given address.
type mutableRange struct {
	addr address
	new  []byte
}

// implements `mutate()` from the `mutable` interface.
func (mut *mutableRange) mutate(b []byte) {
	offset := mut.addr.fullOffset()
	for i, value := range mut.new {
		b[offset+i] = value
	}
}

// sets the interval between beeps when low on hearts
func (rom *romState) setHeartBeepInterval(heartBeepInterval int) {
	mutable := rom.codeMutables["heartBeepInterval"]
	switch heartBeepInterval {
	case HEART_BEEP_HALF:
		mutable.new = []byte{0x3f * 2}
	case HEART_BEEP_QUARTER:
		mutable.new = []byte{0x3f * 4}
	case HEART_BEEP_DISABLED:
		mutable.new = []byte{0x00, 0xc9} // c9 => Unconditional return
	}
}

// sets the amount of required essences to get maku seed
func (rom *romState) setRequiredEssences(requiredEssences int) {
	if requiredEssences >= 8 {
		return
	}

	giveMakuTreeScriptAddr := rom.codeMutables["makuStageEssence8"].new
	for i := 7; i >= requiredEssences; i-- {
		mutableName := "makuStageEssence" + strconv.Itoa(i)
		rom.codeMutables[mutableName].new = giveMakuTreeScriptAddr
	}
}

func (rom *romState) setArchipelagoSlotName(slotName string) {
	slotNameAsBytes := []byte(slotName)
	for i := 0; i < 0x40; i++ {
		if i < len(slotNameAsBytes) {
			rom.data[0xFFFC0+i] = slotNameAsBytes[i]
		} else {
			rom.data[0xFFFC0+i] = 0x00
		}
	}
}

func (rom *romState) setOldManRupeeValues(values map[string]int) {
	valuesArray := rom.codeMutables["oldManRupeeValues"]
	giveTakeArray := rom.codeMutables["oldManGiveTake"]

	OLD_MEN_OFFSETS := []string{
		"old man in goron mountain",
		"old man near blaino",
		"old man near d1",
		"old man near western coast house",
		"old man in horon",
		"old man near d6",
		"old man near holly's house",
		"old man near mrs. ruul",
	}

	for i, name := range OLD_MEN_OFFSETS {
		value, ok := values[name]
		if !ok {
			continue
		}

		absValue := value
		if value < 0 {
			absValue *= -1
		}
		valuesArray.new[i] = RUPEE_VALUES[absValue]

		i *= 2
		if absValue == value {
			giveTakeArray.new[i] = 0x72 // Give
		} else {
			giveTakeArray.new[i] = 0x88 // Take
		}
		giveTakeArray.new[i+1] = 0x74
	}
}

func (rom *romState) setCharacterSprite(sprite string, palette string) error {
	if sprite != "link" {
		exeDir, err := os.Executable()
		if err != nil {
			return err
		}

		dirName := filepath.Dir(exeDir)
		data, err := os.ReadFile(filepath.Join(dirName, "sprites", sprite+".bin"))
		if err != nil {
			return err
		}

		copy(rom.data[0x68000:], data)
	}

	if palette != "green" {
		if paletteByte, ok := PALETTE_BYTES[palette]; ok {
			// Link in-game
			for addr := 0x141cc; addr <= 0x141de; addr += 0x2 {
				rom.data[addr] |= paletteByte
			}

			// Link standing still in file select (fileSelectDrawLink:@sprites0)
			rom.data[0x8d46] |= paletteByte
			rom.data[0x8d4a] |= paletteByte

			// Link animated in file select (@sprites1 & @sprites2)
			rom.data[0x8d4f] |= paletteByte
			rom.data[0x8d53] |= paletteByte
			rom.data[0x8d58] |= paletteByte
			rom.data[0x8d5c] |= paletteByte
		}
	}

	return nil
}

// sets the sequence of seasons + directions required to reach the pedestal in seasons' lost woods
func (rom *romState) setLostWoodsPedestalSequence(sequence [8]byte) {
	builder := new(strings.Builder)
	for i := 0; i < 4; i++ {
		seasonByte := sequence[i*2]
		directionByte := sequence[i*2+1]

		seasonStr := ""
		switch seasonByte {
		case SEASON_SPRING:
			seasonStr = "\x02\xde"
		case SEASON_SUMMER:
			seasonStr = "S\x04\xbc"
		case SEASON_AUTUMN:
			seasonStr = "A\x05\x25"
		case SEASON_WINTER:
			seasonStr = "\x03\x7e"
		}

		directionStr := ""
		switch directionByte {
		case DIRECTION_UP:
			directionStr = "\x03\x01"
		case DIRECTION_RIGHT:
			directionStr = " \x04\x31"
		case DIRECTION_DOWN:
			directionStr = " south"
		case DIRECTION_LEFT:
			directionStr = " \x05\x1e"
		}

		builder.WriteString(seasonStr + directionStr)
		if i != 3 {
			builder.WriteString("\x01")
		} else {
			builder.WriteString("\x00")
		}
		mutableName := "lostWoodsItemSequence" + strconv.Itoa(i+1)
		rom.codeMutables[mutableName].new = []byte{byte(seasonByte), byte(directionByte)}
	}
	rom.codeMutables["lostWoodsPhonographText"].new = []byte(builder.String())
}

// get a collated map of all mutables.
/*
func (rom *romState) getAllMutables() map[string]mutable {
	allMutables := make(map[string]mutable)
	for k, v := range rom.itemSlots {
		addMutOrPanic(allMutables, k, v)
	}
	for k, v := range rom.treasures {
		addMutOrPanic(allMutables, k, v)
	}
	for k, v := range rom.codeMutables {
		addMutOrPanic(allMutables, k, v)
	}
	return allMutables
}

// if the mutable does not exist in the map, add it. if it already exists,
// panic.
func addMutOrPanic(m map[string]mutable, k string, v mutable) {
	if _, ok := m[k]; ok {
		panic("duplicate mutable key: " + k)
	}
	m[k] = v
}
*/

// returns the name of a mutable that covers the given address, or an empty
// string if none is found.
/*
func (rom *romState) findAddr(bank byte, addr uint16) string {
	muts := rom.getAllMutables()
	offset := (&address{bank, addr}).fullOffset()

	for name, mut := range muts {
		switch mut := mut.(type) {
		case *mutableRange:
			if offset >= mut.addr.fullOffset() &&
				offset < mut.addr.fullOffset()+len(mut.new) {
				return name
			}
		case *itemSlot:
			for _, addrs := range [][]address{mut.idAddrs, mut.subidAddrs} {
				for _, addr := range addrs {
					if offset == addr.fullOffset() {
						return name
					}
				}
			}
		case *treasure:
			if offset >= mut.addr.fullOffset() &&
				offset < mut.addr.fullOffset()+4 {
				return name
			}
		default:
			panic("unknown type for mutable: " + name)
		}
	}

	return ""
}
*/
