package randomizer

import (
	"fmt"
)

// an item slot (chest, gift, etc). it references room data and treasure data.
type itemSlot struct {
	treasure                         *treasure
	idAddrs, subidAddrs              []address
	group, room, collectMode, player byte
	moreRooms                        []uint16 // high = group, low = room
	mapTile                          byte     // overworld map coords, yx
}

// implementes `mutate` from the `mutable` interface.
func (mut *itemSlot) mutate(b []byte) {
	for _, addr := range mut.idAddrs {
		b[addr.fullOffset()] = mut.treasure.id
	}
	for _, addr := range mut.subidAddrs {
		b[addr.fullOffset()] = mut.treasure.subid
	}
}

// raw slot data loaded from yaml.
type rawSlot struct {
	// required
	Treasure string
	Room     uint16

	// required if not == low byte of room or in dungeon.
	MapTile *byte

	// pick one, or default to chest
	Addr        rawAddr // for id, then subid
	ReverseAddr rawAddr // for subid, then id
	Ids, SubIds []rawAddr

	// optional override
	Collect string

	// optional additional rooms
	MoreRooms []uint16
}

// like address, but has exported fields for loading from yaml.
type rawAddr struct {
	Bank   byte
	Offset uint16 `yaml:"addr"`
}

// return a map of slot names to slot data. if romState.data is nil, only
// "static" data is loaded.
func (rom *romState) loadSlots() map[string]*itemSlot {
	raws := make(map[string]*rawSlot)

	filename := fmt.Sprintf("romdata/%s_slots.yaml", gameNames[rom.game])
	if err := ReadEmbeddedYaml(filename, raws); err != nil {
		panic(err)
	}

	m := make(map[string]*itemSlot)
	for name, raw := range raws {
		if raw.Room == 0 {
			panic(name + " room is zero")
		}

		slot := &itemSlot{
			treasure:  rom.treasures[raw.Treasure],
			group:     byte(raw.Room >> 8),
			room:      byte(raw.Room),
			moreRooms: raw.MoreRooms,
		}

		// unspecified map tile = assume overworld
		if raw.MapTile == nil && rom.data != nil {
			if slot.group > 2 || (slot.group == 2 && (slot.room&0x0f > 0x0d || slot.room&0xf0 > 0xd0)) {
				// nope, definitely not overworld.
				panic(fmt.Sprintf("invalid room for %s: %04x", name, raw.Room))
			}
			slot.mapTile = slot.room
		} else if raw.MapTile != nil {
			slot.mapTile = *raw.MapTile
		}

		if raw.Collect == "" {
			panic("no collect mode for slot " + name)
		} else if mode, ok := collectModes[raw.Collect]; ok {
			slot.collectMode = mode
		} else {
			panic("collect mode not found: " + raw.Collect)
		}

		if raw.Addr != (rawAddr{}) {
			slot.idAddrs = []address{{raw.Addr.Bank, raw.Addr.Offset}}
			slot.subidAddrs = []address{{raw.Addr.Bank, raw.Addr.Offset + 1}}
		} else if raw.ReverseAddr != (rawAddr{}) {
			slot.idAddrs = []address{{
				raw.ReverseAddr.Bank, raw.ReverseAddr.Offset + 1}}
			slot.subidAddrs = []address{{
				raw.ReverseAddr.Bank, raw.ReverseAddr.Offset}}
		} else if raw.Ids != nil {
			slot.idAddrs = make([]address, len(raw.Ids))
			for i, id := range raw.Ids {
				slot.idAddrs[i] = address{id.Bank, id.Offset}
			}

			// allow absence of subids, only because of seed trees
			if raw.SubIds != nil {
				slot.subidAddrs = make([]address, len(raw.SubIds))
				for i, subid := range raw.SubIds {
					slot.subidAddrs[i] = address{subid.Bank, subid.Offset}
				}
			}
		} else if rom.data != nil {
			isSpecialChest := map[string]bool{
				"chest in master diver's cave": true,
				"mt. cucco, talon's cave":      true,
				"d7 bombed wall chest":         true,
			}

			if slot.collectMode == collectModes["chest"] || isSpecialChest[name] {
				// try to get chest data for room
				addr := getChestAddr(rom.data, rom.game, slot.group, slot.room)
				if addr == (address{}) {
					panic("chest addr not found for " + name)
				}

				slot.idAddrs = []address{{addr.bank, addr.offset}}
				slot.subidAddrs = []address{{addr.bank, addr.offset + 1}}
			}
		}

		m[name] = slot
	}

	return m
}

// returns the full offset of a room's chest's two-byte entry in the rom.
// returns a zero addr if no chest data is found.
func getChestAddr(b []byte, game int, group, room byte) address {
	ptr := sora(game, address{0x15, 0x4f6c}, address{0x16, 0x5108}).(address)
	ptr.offset += uint16(group) * 2
	ptr.offset = uint16(b[ptr.fullOffset()]) + uint16(b[ptr.fullOffset()+1])*0x100

	for {
		info := b[ptr.fullOffset()]
		if info == 0xff {
			break
		}

		chest_room := b[ptr.fullOffset()+1]
		if chest_room == room {
			ptr.offset += 2
			return ptr
		}

		ptr.offset += 4
	}

	return address{}
}
