package rom

// treasure sprites corresponding to byte four of a treasure entry.
// i could use iota or make a []string instead, but i want explicit numbers.
// the gaps are all green sprites of the left half of impa.
// the rest of the values after the end are just more phonographs.
const (
	spriteRupeeSmall  = 0x00
	spriteRupeeMedium = 0x01
	spriteRupeeLarge  = 0x02
	spriteOreChunk    = 0x03
	spriteLines       = 0x04
	spriteBomb        = 0x05
	spriteEmberSeed   = 0x06
	spriteScentSeed   = 0x07
	spritePegasusSeed = 0x08
	spriteGaleSeed    = 0x09
	spriteMysterySeed = 0x0a
	spriteLines2      = 0x0b
	spriteLines3      = 0x0c
	spriteGashaSeed   = 0x0d
	spriteRing        = 0x0e

	spriteSwordL1           = 0x10
	spriteSwordL2           = 0x11
	spriteSwordL3           = 0x12
	spriteShieldL1          = 0x13
	spriteShieldL2          = 0x14
	spriteShieldL3          = 0x15
	spriteFeatherL1         = 0x16
	spriteFeatherL2         = 0x17
	spriteMagnetGlove       = 0x18
	spriteBracelet          = 0x19
	spriteFoolsOre          = 0x1a
	spriteShovel            = 0x1b
	spriteBoomerangL1       = 0x1c
	spriteBoomerangL2       = 0x1d
	spriteRod               = 0x1e
	spriteSwitchHook        = 0x1f
	spriteSatchel           = 0x20
	spriteSlingshotL1       = 0x21
	spriteSlingshotL2       = 0x22
	spriteStrangeFlute      = 0x23
	spriteBombchu           = 0x24
	spriteBiggoronSword     = 0x25
	spriteMastersPlaque     = 0x26
	spriteRupeeSmallGreen   = 0x28
	spriteRupeeSmallCyan    = 0x29
	spriteRupeeSmallYellow  = 0x2a
	spriteRupeeMediumBlue   = 0x2b
	spriteRupeeMediumYellow = 0x2c
	spriteRupeeLargeBlue    = 0x2d
	spriteRupeeLargeYellow  = 0x2e
	spriteOreChunkYellow    = 0x2f
	spriteHalfPotion        = 0x30
	spriteFlippers          = 0x31
	spriteInverseFlipper    = 0x32
	spriteRingBoxL1         = 0x33
	spriteRingBoxL2         = 0x34
	spriteRingBoxL3         = 0x35
	spriteRoundJewel        = 0x36
	spritePyramidJewel      = 0x37
	spriteSquareJewel       = 0x38
	spriteXShapedJewel      = 0x39
	spritePieceOfHeart      = 0x3a
	spriteHeartContainer    = 0x3b

	spriteMap          = 0x40
	spriteCompass      = 0x41
	spriteSmallKey     = 0x42
	spriteBossKey      = 0x43
	spriteGnarledKey   = 0x44
	spriteFloodgateKey = 0x45
	spriteDragonKey    = 0x46
	spriteMakuSeed     = 0x47

	// added these sprites:
	spriteMembersCard = 0x48
	spriteTreasureMap = 0x49

	spriteSpringBanana = 0x54
	spriteRickysGloves = 0x55
	spriteBombFlower   = 0x56
	spriteStarOre      = 0x57
	spriteBlueOre      = 0x58
	spriteRedOre       = 0x59
	spriteHardOre      = 0x5a
	spriteRustyBell    = 0x5b
	spritePiratesBell  = 0x5c
	spriteBlueGloves   = 0x5d

	spriteMakuSeed2 = 0x5f
	spriteEssence1  = 0x60
	spriteEssence2  = 0x61
	spriteEssence3  = 0x62
	spriteEssence4  = 0x63
	spriteEssence5  = 0x64
	spriteEssence6  = 0x65
	spriteEssence7  = 0x66
	spriteEssence8  = 0x67

	spriteCuccodex     = 0x70
	spriteLonLonEgg    = 0x71
	spriteGhastlyDoll  = 0x72
	spriteIronPot      = 0x73
	spriteLavaSoup     = 0x74
	spriteGoronVase    = 0x75
	spriteFish         = 0x76
	spriteMegaphone    = 0x77
	spriteMushroom     = 0x78
	spriteWoodenBird   = 0x79
	spriteEngineGrease = 0x7a
	spritePhonograph   = 0x7b
)

// first two bytes determine sprite; final one determines graphics flags.

var narrowItemGfx = map[string]int{
	"ring":             0x5d0811,
	"gasha seed":       0x5d0a11,
	"ring box":         0x5d1401, // L-1 is 01, L-2 is 11, L-3 is 21
	"gnarled root key": 0x5e0e51,
	"floodgate key":    0x5e1041,
	"dragon key":       0x5e1211,
	"satchel 1":        0x5f0051,
	"satchel 2":        0x5f0050,
	"slingshot L-1":    0x5f0241,
	"slingshot L-2":    0x5f0451,
	"magnet gloves":    0x5f1021,
	"sword L-1":        0x600001,
	"sword L-2":        0x600251,
	"sword L-3":        0x600441,
	"shield L-1":       0x600601,
	"shield L-2":       0x600851,
	"shield L-3":       0x600a41,
	"feather L-1":      0x600c41,
	"feather L-2":      0x600e51,
	"rod":              0x601021,
	"winter":           0x601021,
	"spring":           0x601021,
	"summer":           0x601021,
	"autumn":           0x601021,
	"bracelet":         0x601251,
	"fool's ore":       0x601401,
	"shovel":           0x601641,
	"boomerang L-1":    0x601851,
	"boomerang L-2":    0x601a41,
	"bombs":            0x601c41,
	"round jewel":      0x650400,
	"pyramid jewel":    0x650620,
	"square jewel":     0x650810,
	"x-shaped jewel":   0x650a30,
	"blue ore":         0x660410,
	"red ore":          0x660420,

	// don't slot these in scenes
	/*
		"ember seeds":   0x5f0621,
		"scent seeds":   0x5f0831,
		"pegasus seeds": 0x5f0a11,
		"gale seeds":    0x5f0c11,
		"mystery seeds": 0x5f0e01,
	*/
}

var wideItemGfx = map[string]int{
	"heart refill":     0x5c0257,
	"member's card":    0x5d0c13,
	"master's plaque":  0x5d0c33,
	"piece of heart":   0x5d1022,
	"rare peach stone": 0x5d1026, // piece of heart in subrosian market
	"strange flute":    0x5f1603,
	"ricky's gloves":   0x641c53,
	"ribbon":           0x650c23,
	"spring banana":    0x651033,
	"treasure map":     0x651433,
	"rusty bell":       0x651823,
	"star ore":         0x660033,
}

// union of narrow and wide item graphics, initialized in init()
var itemGfx = map[string]int{}

// CanSlotInScene returns true iff the named item can display correctly in the
// rod of seasons and noble sword scenes.
func CanSlotInScene(itemName string) bool {
	return narrowItemGfx[itemName] != 0
}

// CanSlotInShop returns true iff the named item can display correctly in the
// Horon Village shop.
func CanSlotInShop(itemName string) bool {
	return itemGfx[itemName] != 0
}

// CanSlotInMarket returns true iff the named item can display correctly in the
// Subrosian market.
func CanSlotInMarket(itemName string) bool {
	return wideItemGfx[itemName] != 0
}
