package randomizer

import (
	"fmt"
	"strconv"
    "strings"
)

// an instance of ROM data that can be changed by the randomizer.
type mutable interface {
	mutate([]byte)      // change ROM bytes
	check([]byte) error // verify that the mutable matches the ROM
}

// a length of mutable bytes starting at a given address.
type mutableRange struct {
	addr     address
	old, new []byte
}

// implements `mutate()` from the `mutable` interface.
func (mut *mutableRange) mutate(b []byte) {
	offset := mut.addr.fullOffset()
	for i, value := range mut.new {
		b[offset+i] = value
	}
}

// implements `check()` from the `mutable` interface.
func (mut *mutableRange) check(b []byte) error {
	offset := mut.addr.fullOffset()
	for i, value := range mut.old {
		if b[offset+i] != value {
			return fmt.Errorf("expected %x at %x; found %x",
				mut.old[i], offset+i, b[offset+i])
		}
	}
	return nil
}

// sets treewarp on or off in the modified ROM. By default, it is on.
func (rom *romState) setTreewarp(treewarp bool) {
	mut := rom.codeMutables["treeWarp"]
	mut.new[5] = byte(ternary(treewarp, 0x28, 0x18).(int)) // jr z / jr
}

// sets the interval between beeps when low on hearts
func (rom *romState) setHeartBeepInterval(heartBeepInterval int) {
	mutable := rom.codeMutables["heartBeepInterval"]
    switch heartBeepInterval {
    case HEART_BEEP_HALF:     mutable.new = []byte{0x3f * 2}
    case HEART_BEEP_QUARTER:  mutable.new = []byte{0x3f * 4}
    case HEART_BEEP_DISABLED: mutable.new = []byte{0x00, 0xc9}  // c9 => Unconditional return
    }
}

// sets the damage dealt by fool's ore in seasons
func (rom *romState) setFoolsOreDamage(foolsOreDamage int) {
	foolsOreDamage *= -1
	rom.codeMutables["foolsOreDamage"].new = []byte{byte(foolsOreDamage)}
}

// sets the amount of required essences to get maku seed
func (rom *romState) setRequiredEssences(requiredEssences int) {
	if requiredEssences >= 8 {
		return
	}

    giveMakuTreeScriptAddr := rom.codeMutables["makuStageEssence8"].new
    for i:=7 ; i >= requiredEssences; i-- {
        mutableName := "makuStageEssence" + strconv.Itoa(i)
        rom.codeMutables[mutableName].new = giveMakuTreeScriptAddr
    }
}

func (rom *romState) setArchipelagoSlotName(slotName string) {
	slotNameAsBytes := []byte(slotName)
    for i := 0 ; i<0x40 ; i++ {
        if i < len(slotNameAsBytes) {
            rom.data[0xFFFC0 + i] = slotNameAsBytes[i]
        } else {
            rom.data[0xFFFC0 + i] = 0x00
        }        
    }
}

func (rom *romState) setOldManRupeeValues(values map[string]int) {
	valuesArray := rom.codeMutables["oldManRupeeValues"]
	giveTakeArray := rom.codeMutables["oldManGiveTake"]

	funcGiveAddr := giveTakeArray.new[0:2]
	funcTakeAddr := giveTakeArray.new[14:16]

	var OLD_MEN_OFFSETS map[string]int

	if rom.game == GAME_SEASONS {
		OLD_MEN_OFFSETS = map[string]int{
			"old man in goron mountain": 0,
			"old man near blaino": 1,
			"old man near d1": 2,
			"old man near western coast house": 3,
			"old man in horon": 4,
			"old man in treehouse": 5,
			"old man near holly's house": 6,
			"old man near mrs. ruul": 7,
		}
	}

	for key, value := range values {
		offset := OLD_MEN_OFFSETS[key]

		absValue := value
		if value < 0 {
			absValue *= -1
		}
		valuesArray.new[offset] = RUPEE_VALUES[absValue]

		offset *= 2
		if absValue == value {  // Give
			giveTakeArray.new[offset] = funcGiveAddr[0]
			giveTakeArray.new[offset+1] = funcGiveAddr[1]
		} else { 				// Take
			giveTakeArray.new[offset] = funcTakeAddr[0]
			giveTakeArray.new[offset+1] = funcTakeAddr[1]
		}
	}
}

// sets the sequence of seasons + directions required to reach the pedestal in seasons' lost woods
func (rom *romState) setLostWoodsPedestalSequence(sequence [8]byte) {
	builder := new(strings.Builder)
    for i:=0 ; i<4 ; i++ {
        seasonByte := sequence[i*2]
		directionByte := sequence[i*2+1]
		
        seasonStr := ""
        switch seasonByte {
        case SEASON_SPRING:  seasonStr = "\x02\xde"
        case SEASON_SUMMER:  seasonStr = "S\x04\xbc"
        case SEASON_AUTUMN:  seasonStr = "A\x05\x25"
        case SEASON_WINTER:  seasonStr = "\x03\x7e"
        }

        directionStr := ""
        switch directionByte {
        case DIRECTION_UP:    directionStr = "\x03\x01"
        case DIRECTION_RIGHT: directionStr = " \x04\x31"
        case DIRECTION_DOWN:  directionStr = " south"
        case DIRECTION_LEFT:  directionStr = " \x05\x1e"
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

// sets the natzu region based on a companion number 1 to 3.
func (rom *romState) setAnimal(companion int) {
	rom.codeMutables["romAnimalRegion"].new =
		[]byte{byte(companion + 0x0a)}
}

// key = area name (as in asm/vars.yaml), id = season index (spring -> winter).
func (rom *romState) setSeason(key string, id byte) {
	rom.codeMutables[key].new[0] = id
}

// get a collated map of all mutables.
func (rom *romState) getAllMutables() map[string]mutable {
	allMutables := make(map[string]mutable)
	for k, v := range rom.itemSlots {
		if v.treasure == nil {
			panic(fmt.Sprintf("treasure for %s is nil", k))
		}
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

// returns the name of a mutable that covers the given address, or an empty
// string if none is found.
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
