# Zelda Oracles Randomizer

> Summary of different randomizer versions and links to their web interfaces:
>
> - [This version](https://cemulate.github.io/oracles-randomizer-web/), which supports multiworld
> - [vinheim3's version](https://cemulate.github.io/oracles-randomizer-web/), which supports full entrance randomization
> - [Stewmath's version](https://oosarando2.zeldahacking.net/), which supports keysanity and cross items
> - [karafruit's version](https://oracles-dev.gwaa.kiwi/generate), which supports Stewmath's features plus some goodies

This program reads a Zelda: Oracle of Seasons or Oracle of Ages ROM (US
versions only), shuffles the locations of (most) items and mystical seeds, and
writes the modified ROM to a new file. In Seasons, the default seasons for each
area are also randomized. Most arbitrary overworld checks for essences and
other game flags are removed, so the dungeons and other checks can be done in
any order that the randomized items facilitate. However, you do need to collect
all 8 essences to get the Maku Seed and finish the game.


## Usage

You probably want the
[web interface](https://cemulate.github.io/oracles-randomizer-web),
contributed by [cemulate](https://github.com/cemulate).

Otherwise, there are three ways to use the randomizer:

1. Place the randomizer in the same directory as your vanila ROM(s) (or vice
   versa), and run it. The randomizer will automatically find your vanilla
   ROM(s) and prompt for further options.
2. In Windows, drag your vanilla ROM onto the executable. Same deal as above,
   except that the ROM and randomizer don't have to be in the same folder.
3. Use the command line. Type `./oracles-randomizer -h` to view the usage
   summary.

You may also be interested in
[Oracles Randomizer Extras](https://jangler.github.io/oracles-randomizer-extras/).


## Download

You can download executables for Windows, macOS, and Linux from the
[releases](https://github.com/dinopony/oracles-archipelago-patcher/releases) page. Don't
use the "Download ZIP" link on the main page; that only contains the source
code. The download also contains a rudimentary location checklist and item
tracker. If you're looking for a more detailed item and map tracker,
[EmoTracker](https://emotracker.net/) has a pack developed by Herreteman.

See
[contributing.md](https://github.com/dinopony/oracles-archipelago-patcher/blob/master/doc/contributing.md)
for instructions on building the randomizer from source.


## Randomization notes

General details common to both games:

- Items and chests are randomized, with these exceptions:
    - Renewable shop and business scrub items (bombs, shield, hearts, etc.)
	- Gasha seeds and pieces of heart outside of chests
	- NPCs that give non-progression items in the vanilla game
	- Gasha nut contents
	- Fixed drops (from bushes, pots, etc.)
	- Maple drops
	- Linked secrets
- Mystical seed trees are randomized, with no more than two trees of each type.
  Items that use seeds for ammunition start with the type of seed that's on the
  Horon Village or Lynna City tree.
- For items that have two levels, the first you obtain will be L-1, and the
  second will be L-2, regardless of the order in which you obtain them. The L-2
  shield is an exception.
- There is one flute in the game for a random animal companion, and it's
  identified and usable as soon as you get it. Only the 150-rupee item in the
  shop is randomized; the other two usual means of getting a strange flute
  don't give anything special. The animal companion regions (Natzu in Seasons
  and Nuun in Ages) match whatever flute is in the seed.
- Rings are instantly appraised when you get them, and the ring list can be
  accessed from the inventory ring box icon. For convenience, the L-3 ring box
  is given at the start. The punch rings can be used with only one equip slot
  empty.
- Select+right on the file select screen toggles music. Select+left on the file
  select screen toggles between GBC palettes (default) and lighter GBA
  palettes; this will only have an effect if you're playing on or emulating a
  GBA.
- If tree warp is enabled, holding start while closing the map screen outdoors
  warps to the seed tree in Horon Village or Lynna City. Tree warp comes with
  no warranty and is not supported as a "feature", so think carefully before
  using it.
- If hard difficulty is enabled, speedrun-level tricks may be required to
  complete the game. Use normal difficulty if you just want to do a casual
  playthrough!

For game-specific notes on randomization and logic, see
[seasons_notes.md](https://github.com/dinopony/oracles-archipelago-patcher/blob/master/doc/seasons_notes.md)
and
[ages_notes.md](https://github.com/dinopony/oracles-archipelago-patcher/blob/master/doc/ages_notes.md).

See
[multiworld.md](https://github.com/dinopony/oracles-archipelago-patcher/blob/master/doc/multiworld.md)
for information on multiworld seeds.

See
[plan.md](https://github.com/dinopony/oracles-archipelago-patcher/blob/master/doc/plan.md)
for information on plando generation.


## FAQ

**Q: Is there a place to discuss the randomizer?**

A: Yes, the [Oracles Discord server](https://discord.gg/pyBEbz5). The server is
mainly focused on speedrunning, but randomizer-specific channels exist as well.

**Q: I found a problem. What do I do?**

A: Open an issue about it on GitHub or bring it up in a randomizer channel in
the Oracles discord. Provide your seed's log file either way.

**Q: Will you make a cross-game randomizer that combines Ages and Seasons into
one ROM?**

A: no

**Q: Can I at least do a linked game?**

A: You can try, as long as you're not doing multiworld, but the consensus seems
to be that it's not worthwhile. Linked content is unrandomized and sometimes
inaccessible. Linked Ages seeds also have a chance to be uncompletable due to
how Sea of Storms works.

**Q: What emulator would you recommend for playing the randomizer?**

A: If you want to play multiworld, you must use Bizhawk. BGB and mGBA are good
choices otherwise. Avoid VisualBoyAdvance and its variants; they have serious
emulation bugs that can cause crashes and other problems.


## Thanks to:

- Stewmath for [oracles-disasm](https://github.com/Stewmath/oracles-disasm) and
  additional code.
- Herreteman, dragonc0, Phoenomenom714, and jaysee87 for help with logic,
  playtesting, design, and "customer support".
- Everyone who helped playtest prerelease versions of the randomizer.
