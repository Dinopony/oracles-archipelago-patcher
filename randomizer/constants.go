package randomizer

const VERSION = "0.9a"

const (
	gameNil = iota
	gameAges = 1
	gameSeasons = 2
)

var gameNames = map[int]string{
	gameNil:     "nil",
	gameAges:    "ages",
	gameSeasons: "seasons",
}

const (
	COMPANION_RICKY   = 1
	COMPANION_DIMITRI = 2
	COMPANION_MOOSH   = 3
)
