local M = {}

-- This module uses bgb/no$gmb syntax and enforces a strict style: numbers are
-- undecorated (no `$`, etc.) and hexadecimal, `a,` is required in mnemonics
-- that feature it, spaces do not appear after `,`s, parens are used for
-- "dereferencing" memory, and all keywords are lower-case. User-defined
-- symbols such as labels are case-sensitive. A label and instruction cannot
-- appear on the same line.
--
-- The characters in `;*#` all begin inline comments, although instruction
-- delimiter status overrides comment character status in the `compile()`
-- function.
--
-- "Local" labels (the only kind) start with a `.` and can be referenced by
-- relative jumps. Decompilation automatically generates labels for relative
-- jump destinations.

-- lua tables are 1-indexed, but think of this as a map and not an array. it
-- can't be iterated over like an array, since it starts at zero and contains
-- nil values.
local mnemonics = {
    'ld bc,d16',      -- 01
    'ld (bc),a',      -- 02
    'inc bc',         -- 03
    'inc b',          -- 04
    'dec b',          -- 05
    'ld b,d8',        -- 06
    'rlca',           -- 07
    'ld (a16),sp',    -- 08
    'add hl,bc',      -- 09
    'ld a,(bc)',      -- 0a
    'dec bc',         -- 0b
    'inc c',          -- 0c
    'dec c',          -- 0d
    'ld c,d8',        -- 0e
    'rrca',           -- 0f
    'stop 0',         -- 10
    'ld de,d16',      -- 11
    'ld (de),a',      -- 12
    'inc de',         -- 13
    'inc d',          -- 14
    'dec d',          -- 15
    'ld d,d8',        -- 16
    'rla',            -- 17
    'jr r8',          -- 18
    'add hl,de',      -- 19
    'ld a,(de)',      -- 1a
    'dec de',         -- 1b
    'inc e',          -- 1c
    'dec e',          -- 1d
    'ld e,d8',        -- 1e
    'rra',            -- 1f
    'jr nz,r8',       -- 20
    'ld hl,d16',      -- 21
    'ldi (hl),a',     -- 22
    'inc hl',         -- 23
    'inc h',          -- 24
    'dec h',          -- 25
    'ld h,d8',        -- 26
    'daa',            -- 27
    'jr z,r8',        -- 28
    'add hl,hl',      -- 29
    'ldi a,(hl)',     -- 2a
    'dec hl',         -- 2b
    'inc l',          -- 2c
    'dec l',          -- 2d
    'ld l,d8',        -- 2e
    'cpl',            -- 2f
    'jr nc,r8',       -- 30
    'ld sp,d16',      -- 31
    'ldd (hl),a',     -- 32
    'inc sp',         -- 33
    'inc (hl)',       -- 34
    'dec (hl)',       -- 35
    'ld (hl),d8',     -- 36
    'scf',            -- 37
    'jr c,r8',        -- 38
    'add hl,sp',      -- 39
    'ldd a,(hl)',     -- 3a
    'dec sp',         -- 3b
    'inc a',          -- 3c
    'dec a',          -- 3d
    'ld a,d8',        -- 3e
    'ccf',            -- 3f
    'ld b,b',         -- 40
    'ld b,c',         -- 41
    'ld b,d',         -- 42
    'ld b,e',         -- 43
    'ld b,h',         -- 44
    'ld b,l',         -- 45
    'ld b,(hl)',      -- 46
    'ld b,a',         -- 47
    'ld c,b',         -- 48
    'ld c,c',         -- 49
    'ld c,d',         -- 4a
    'ld c,e',         -- 4b
    'ld c,h',         -- 4c
    'ld c,l',         -- 4d
    'ld c,(hl)',      -- 4e
    'ld c,a',         -- 4f
    'ld d,b',         -- 50
    'ld d,c',         -- 51
    'ld d,d',         -- 52
    'ld d,e',         -- 53
    'ld d,h',         -- 54
    'ld d,l',         -- 55
    'ld d,(hl)',      -- 56
    'ld d,a',         -- 57
    'ld e,b',         -- 58
    'ld e,c',         -- 59
    'ld e,d',         -- 5a
    'ld e,e',         -- 5b
    'ld e,h',         -- 5c
    'ld e,l',         -- 5d
    'ld e,(hl)',      -- 5e
    'ld e,a',         -- 5f
    'ld h,b',         -- 60
    'ld h,c',         -- 61
    'ld h,d',         -- 62
    'ld h,e',         -- 63
    'ld h,h',         -- 64
    'ld h,l',         -- 65
    'ld h,(hl)',      -- 66
    'ld h,a',         -- 67
    'ld l,b',         -- 68
    'ld l,c',         -- 69
    'ld l,d',         -- 6a
    'ld l,e',         -- 6b
    'ld l,h',         -- 6c
    'ld l,l',         -- 6d
    'ld l,(hl)',      -- 6e
    'ld l,a',         -- 6f
    'ld (hl),b',      -- 70
    'ld (hl),c',      -- 71
    'ld (hl),d',      -- 72
    'ld (hl),e',      -- 73
    'ld (hl),h',      -- 74
    'ld (hl),l',      -- 75
    'halt',           -- 76
    'ld (hl),a',      -- 77
    'ld a,b',         -- 78
    'ld a,c',         -- 79
    'ld a,d',         -- 7a
    'ld a,e',         -- 7b
    'ld a,h',         -- 7c
    'ld a,l',         -- 7d
    'ld a,(hl)',      -- 7e
    'ld a,a',         -- 7f
    'add a,b',        -- 80
    'add a,c',        -- 81
    'add a,d',        -- 82
    'add a,e',        -- 83
    'add a,h',        -- 84
    'add a,l',        -- 85
    'add a,(hl)',     -- 86
    'add a,a',        -- 87
    'adc a,b',        -- 88
    'adc a,c',        -- 89
    'adc a,d',        -- 8a
    'adc a,e',        -- 8b
    'adc a,h',        -- 8c
    'adc a,l',        -- 8d
    'adc a,(hl)',     -- 8e
    'adc a,a',        -- 8f
    'sub b',          -- 90
    'sub c',          -- 91
    'sub d',          -- 92
    'sub e',          -- 93
    'sub h',          -- 94
    'sub l',          -- 95
    'sub (hl)',       -- 96
    'sub a',          -- 97
    'sbc a,b',        -- 98
    'sbc a,c',        -- 99
    'sbc a,d',        -- 9a
    'sbc a,e',        -- 9b
    'sbc a,h',        -- 9c
    'sbc a,l',        -- 9d
    'sbc a,(hl)',     -- 9e
    'sbc a,a',        -- 9f
    'and b',          -- a0
    'and c',          -- a1
    'and d',          -- a2
    'and e',          -- a3
    'and h',          -- a4
    'and l',          -- a5
    'and (hl)',       -- a6
    'and a',          -- a7
    'xor b',          -- a8
    'xor c',          -- a9
    'xor d',          -- aa
    'xor e',          -- ab
    'xor h',          -- ac
    'xor l',          -- ad
    'xor (hl)',       -- ae
    'xor a',          -- af
    'or b',           -- b0
    'or c',           -- b1
    'or d',           -- b2
    'or e',           -- b3
    'or h',           -- b4
    'or l',           -- b5
    'or (hl)',        -- b6
    'or a',           -- b7
    'cp b',           -- b8
    'cp c',           -- b9
    'cp d',           -- ba
    'cp e',           -- bb
    'cp h',           -- bc
    'cp l',           -- bd
    'cp (hl)',        -- be
    'cp a',           -- bf
    'ret nz',         -- c0
    'pop bc',         -- c1
    'jp nz,a16',      -- c2
    'jp a16',         -- c3
    'call nz,a16',    -- c4
    'push bc',        -- c5
    'add a,d8',       -- c6
    'rst 00',         -- c7
    'ret z',          -- c8
    'ret',            -- c9
    'jp z,a16',       -- ca
    'prefix cb',      -- cb
    'call z,a16',     -- cc
    'call a16',       -- cd
    'adc a,d8',       -- ce
    'rst 08',         -- cf
    'ret nc',         -- d0
    'pop de',         -- d1
    'jp nc,a16',      -- d2
    nil,              -- d3
    'call nc,a16',    -- d4
    'push de',        -- d5
    'sub a,d8',       -- d6
    'rst 10',         -- d7
    'ret c',          -- d8
    'reti',           -- d9
    'jp c,a16',       -- da
    nil,              -- db
    'call c,a16',     -- dc
    nil,              -- dd
    'sbc a,d8',       -- de
    'rst 18',         -- df
    'ld (ff00+a8),a', -- e0
    'pop hl',         -- e1
    'ld (ff00+c),a',  -- e2
    nil,              -- e3
    nil,              -- e4
    'push hl',        -- e5
    'and a,d8',       -- e6
    'rst 20',         -- e7
    'add sp,r8',      -- e8
    'jp (hl)',        -- e9
    'ld (a16),a',     -- ea
    nil,              -- eb
    nil,              -- ec
    nil,              -- ed
    'xor a,d8',       -- ee
    'rst 28',         -- ef
    'ld a,(ff00+a8)', -- f0
    'pop af',         -- f1
    'ld a,(ff00+c)',  -- f2
    'di',             -- f3
    nil,              -- f4
    'push af',        -- f5
    'or a,d8',        -- f6
    'rst 30',         -- f7
    'ld hl,sp+r8',    -- f8
    'ld sp,hl',       -- f9
    'ld a,(a16)',     -- fa
    'ei',             -- fb
    nil,              -- fc
    nil,              -- fd
    'cp a,d8',        -- fe
    'rst 38',         -- ff
}
mnemonics[0] = 'nop'

-- if instruction doesn't match in `opcodes`, check this table.
local cb_mnemonics = {
    'rlc c',      -- 01
    'rlc d',      -- 02
    'rlc e',      -- 03
    'rlc h',      -- 04
    'rlc l',      -- 05
    'rlc (hl)',   -- 06
    'rlc a',      -- 07
    'rrc b',      -- 08
    'rrc c',      -- 09
    'rrc d',      -- 0a
    'rrc e',      -- 0b
    'rrc h',      -- 0c
    'rrc l',      -- 0d
    'rrc (hl)',   -- 0e
    'rrc a',      -- 0f
    'rl b',       -- 10
    'rl c',       -- 11
    'rl d',       -- 12
    'rl e',       -- 13
    'rl h',       -- 14
    'rl l',       -- 15
    'rl (hl)',    -- 16
    'rl a',       -- 17
    'rr b',       -- 18
    'rr c',       -- 19
    'rr d',       -- 1a
    'rr e',       -- 1b
    'rr h',       -- 1c
    'rr l',       -- 1d
    'rr (hl)',    -- 1e
    'rr a',       -- 1f
    'sla b',      -- 20
    'sla c',      -- 21
    'sla d',      -- 22
    'sla e',      -- 23
    'sla h',      -- 24
    'sla l',      -- 25
    'sla (hl)',   -- 26
    'sla a',      -- 27
    'sra b',      -- 28
    'sra c',      -- 29
    'sra d',      -- 2a
    'sra e',      -- 2b
    'sra h',      -- 2c
    'sra l',      -- 2d
    'sra (hl)',   -- 2e
    'sra a',      -- 2f
    'swap b',     -- 30
    'swap c',     -- 31
    'swap d',     -- 32
    'swap e',     -- 33
    'swap h',     -- 34
    'swap l',     -- 35
    'swap (hl)',  -- 36
    'swap a',     -- 37
    'srl b',      -- 38
    'srl c',      -- 39
    'srl d',      -- 3a
    'srl e',      -- 3b
    'srl h',      -- 3c
    'srl l',      -- 3d
    'srl (hl)',   -- 3e
    'srl a',      -- 3f
    'bit 0,b',    -- 40
    'bit 0,c',    -- 41
    'bit 0,d',    -- 42
    'bit 0,e',    -- 43
    'bit 0,h',    -- 44
    'bit 0,l',    -- 45
    'bit 0,(hl)', -- 46
    'bit 0,a',    -- 47
    'bit 1,b',    -- 48
    'bit 1,c',    -- 49
    'bit 1,d',    -- 4a
    'bit 1,e',    -- 4b
    'bit 1,h',    -- 4c
    'bit 1,l',    -- 4d
    'bit 1,(hl)', -- 4e
    'bit 1,a',    -- 4f
    'bit 2,b',    -- 50
    'bit 2,c',    -- 51
    'bit 2,d',    -- 52
    'bit 2,e',    -- 53
    'bit 2,h',    -- 54
    'bit 2,l',    -- 55
    'bit 2,(hl)', -- 56
    'bit 2,a',    -- 57
    'bit 3,b',    -- 58
    'bit 3,c',    -- 59
    'bit 3,d',    -- 5a
    'bit 3,e',    -- 5b
    'bit 3,h',    -- 5c
    'bit 3,l',    -- 5d
    'bit 3,(hl)', -- 5e
    'bit 3,a',    -- 5f
    'bit 4,b',    -- 60
    'bit 4,c',    -- 61
    'bit 4,d',    -- 62
    'bit 4,e',    -- 63
    'bit 4,h',    -- 64
    'bit 4,l',    -- 65
    'bit 4,(hl)', -- 66
    'bit 4,a',    -- 67
    'bit 5,b',    -- 68
    'bit 5,c',    -- 69
    'bit 5,d',    -- 6a
    'bit 5,e',    -- 6b
    'bit 5,h',    -- 6c
    'bit 5,l',    -- 6d
    'bit 5,(hl)', -- 6e
    'bit 5,a',    -- 6f
    'bit 6,b',    -- 70
    'bit 6,c',    -- 71
    'bit 6,d',    -- 72
    'bit 6,e',    -- 73
    'bit 6,h',    -- 74
    'bit 6,l',    -- 75
    'bit 6,(hl)', -- 76
    'bit 6,a',    -- 77
    'bit 7,b',    -- 78
    'bit 7,c',    -- 79
    'bit 7,d',    -- 7a
    'bit 7,e',    -- 7b
    'bit 7,h',    -- 7c
    'bit 7,l',    -- 7d
    'bit 7,(hl)', -- 7e
    'bit 7,a',    -- 7f
    'res 0,b',    -- 80
    'res 0,c',    -- 81
    'res 0,d',    -- 82
    'res 0,e',    -- 83
    'res 0,h',    -- 84
    'res 0,l',    -- 85
    'res 0,(hl)', -- 86
    'res 0,a',    -- 87
    'res 1,b',    -- 88
    'res 1,c',    -- 89
    'res 1,d',    -- 8a
    'res 1,e',    -- 8b
    'res 1,h',    -- 8c
    'res 1,l',    -- 8d
    'res 1,(hl)', -- 8e
    'res 1,a',    -- 8f
    'res 2,b',    -- 90
    'res 2,c',    -- 91
    'res 2,d',    -- 92
    'res 2,e',    -- 93
    'res 2,h',    -- 94
    'res 2,l',    -- 95
    'res 2,(hl)', -- 96
    'res 2,a',    -- 97
    'res 3,b',    -- 98
    'res 3,c',    -- 99
    'res 3,d',    -- 9a
    'res 3,e',    -- 9b
    'res 3,h',    -- 9c
    'res 3,l',    -- 9d
    'res 3,(hl)', -- 9e
    'res 3,a',    -- 9f
    'res 4,b',    -- a0
    'res 4,c',    -- a1
    'res 4,d',    -- a2
    'res 4,e',    -- a3
    'res 4,h',    -- a4
    'res 4,l',    -- a5
    'res 4,(hl)', -- a6
    'res 4,a',    -- a7
    'res 5,b',    -- a8
    'res 5,c',    -- a9
    'res 5,d',    -- aa
    'res 5,e',    -- ab
    'res 5,h',    -- ac
    'res 5,l',    -- ad
    'res 5,(hl)', -- ae
    'res 5,a',    -- af
    'res 6,b',    -- b0
    'res 6,c',    -- b1
    'res 6,d',    -- b2
    'res 6,e',    -- b3
    'res 6,h',    -- b4
    'res 6,l',    -- b5
    'res 6,(hl)', -- b6
    'res 6,a',    -- b7
    'res 7,b',    -- b8
    'res 7,c',    -- b9
    'res 7,d',    -- ba
    'res 7,e',    -- bb
    'res 7,h',    -- bc
    'res 7,l',    -- bd
    'res 7,(hl)', -- be
    'res 7,a',    -- bf
    'set 0,b',    -- c0
    'set 0,c',    -- c1
    'set 0,d',    -- c2
    'set 0,e',    -- c3
    'set 0,h',    -- c4
    'set 0,l',    -- c5
    'set 0,(hl)', -- c6
    'set 0,a',    -- c7
    'set 1,b',    -- c8
    'set 1,c',    -- c9
    'set 1,d',    -- ca
    'set 1,e',    -- cb
    'set 1,h',    -- cc
    'set 1,l',    -- cd
    'set 1,(hl)', -- ce
    'set 1,a',    -- cf
    'set 2,b',    -- d0
    'set 2,c',    -- d1
    'set 2,d',    -- d2
    'set 2,e',    -- d3
    'set 2,h',    -- d4
    'set 2,l',    -- d5
    'set 2,(hl)', -- d6
    'set 2,a',    -- d7
    'set 3,b',    -- d8
    'set 3,c',    -- d9
    'set 3,d',    -- da
    'set 3,e',    -- db
    'set 3,h',    -- dc
    'set 3,l',    -- dd
    'set 3,(hl)', -- de
    'set 3,a',    -- df
    'set 4,b',    -- e0
    'set 4,c',    -- e1
    'set 4,d',    -- e2
    'set 4,e',    -- e3
    'set 4,h',    -- e4
    'set 4,l',    -- e5
    'set 4,(hl)', -- e6
    'set 4,a',    -- e7
    'set 5,b',    -- e8
    'set 5,c',    -- e9
    'set 5,d',    -- ea
    'set 5,e',    -- eb
    'set 5,h',    -- ec
    'set 5,l',    -- ed
    'set 5,(hl)', -- ee
    'set 5,a',    -- ef
    'set 6,b',    -- f0
    'set 6,c',    -- f1
    'set 6,d',    -- f2
    'set 6,e',    -- f3
    'set 6,h',    -- f4
    'set 6,l',    -- f5
    'set 6,(hl)', -- f6
    'set 6,a',    -- f7
    'set 7,b',    -- f8
    'set 7,c',    -- f9
    'set 7,d',    -- fa
    'set 7,e',    -- fb
    'set 7,h',    -- fc
    'set 7,l',    -- fd
    'set 7,(hl)', -- fe
    'set 7,a',    -- ff
}
cb_mnemonics[0] = 'rlc b'

-- create opcode lookup tables from mnemonics.
-- entries in the table have the fields:
-- - 'opcode': uint8 code of the instruction
-- - 'pindex': index of the mnemonic's parameter (or nil)
-- - 'arity': number of bytes of arguments the instruction takes
-- keys have the argument metavariable stripped from the mnemonic.
local ops = {}

for opcode, mnemonic in pairs(mnemonics) do
    local arg8 = string.find(mnemonic, '%a8')
    local arg16 = string.find(mnemonic, '%a16')
    mnemonic = string.gsub(mnemonic, '%a[816]+', '')

    if arg8 then
        ops[mnemonic] = {opcode = opcode, pindex = arg8, arity = 1}
    elseif arg16 then
        ops[mnemonic] = {opcode = opcode, pindex = arg16, arity = 2}
    else
        ops[mnemonic] = {opcode = opcode, pindex = nil, arity = 0}
    end
end

-- cb prefix opcodes work slightly differently. the table entries are
-- opcode-only, since all cb instructions are two bytes and take no arguments.
local cb_opcodes = {}

for code, mnemonic in pairs(cb_mnemonics) do
    cb_opcodes[mnemonic] = code
end

-- strips indent and comment from line, if present.
local function strip_line(line)
    line = string.gsub(line, '^%s+', '') -- indent
    line = string.gsub(line, '%s*[;*#].*', '') -- comment
    return line
end

-- assembles a sequence of comma-separated numeric literals.
local function compile_literals(bits, line, defs)
    local values = {}

    local arg_string = string.sub(line, 4) -- strip '^db '
    for entry in string.gmatch(arg_string, '[^,]+') do
        local value = 0

        -- just use a placeholder if defs isn't defined (i.e. first pass)
        if defs then
            if defs[entry] then
                value = defs[entry]
            else
                if not string.match(entry, '^%x+$') then
                    error(line .. ': define not found: ' .. entry)
                end
                value = tonumber(entry, 16)
            end
        end

        if value < 0 or value >= math.pow(2, bits) then
            error(line .. ': invalid argument: ' .. entry)
        end
        table.insert(values, value)
    end

    if #values == 0 then
        error(line .. ': missing argument')
    end

    -- convert values to big-endian byte sequences
    local bytes = {}
    for _, value in ipairs(values) do
        for i = 1, bits / 8 do
            table.insert(bytes, value % 0x100)
            value = math.floor(value / 0x100)
        end
    end
    return unpack(bytes)
end

-- Creates a sequence of literal bytes.
local function compile_db(line, defs)
    return compile_literals(8, line, defs)
end

-- Creates a sequence of big-endian words.
local function compile_dw(line, defs)
    return compile_literals(16, line, defs)
end

-- Associates a symbol with a constant numeric value.
local function add_define(line, defs)
    if not defs then
        return unpack({})
    end

    local symbol, value = string.match(line, '^define ([%a_][%w_.]*),(%x+)$')
    if not symbol then
        error(line .. ': invalid define')
    end
    defs[symbol] = tonumber(value, 16)

    return unpack({})
end

-- as `compile_line_to_bytes()`, but with a preidentified operation and string
-- argument.
local function compile_op_with_arg(line, op, word, offset, labels, defs)
    -- labels are already numbers (or should be)
    local arg = tonumber(word, 16)
    local is_jr = string.match(line, '^jr')
    if is_jr then
        -- jr ops can only use local labels
        if labels and labels[word] then
            -- adjust for relative address
            arg = (labels[word] - offset - 2) % 0x100
        end
    else
        if defs and defs[word] then
            arg = defs[word]
        end
    end

    if not arg then
        -- no label table = first pass; use placeholder zero
        if labels then
            if is_jr then
                error(line .. ': local label not found: ' .. word)
            else
                error(line .. ': define not found: ' .. word)
            end
        else
            arg = 0
        end
    end

    -- make sure argument is in a valid range for the instruction
    if arg < 0 or arg >= math.pow(2, op.arity * 8) then
        error(line .. ': invalid argument: ' .. arg)
    end

    -- return appropriate number of bytes
    if op.arity == 1 then
        return op.opcode, arg
    else
        return op.opcode, arg % 0x100, math.floor(arg / 0x100)
    end
end

-- functions invoked for keyword commands. args are (line, defs).
local keyword_funcs = {
    db = compile_db,
    dw = compile_dw,
    define = add_define,
}

-- as `compile_line()`, but returns a series of bytes instead of a byte string.
local function compile_line_to_bytes(line, offset, labels, defs)
    line = strip_line(line)

    -- first try matching a keyword
    local keyword = string.match(line, '^%a+')
    if keyword_funcs[keyword] then
        return keyword_funcs[keyword](line, defs)
    end

    -- then try raw index (works against nullary instructions)
    if ops[line] then
        return ops[line].opcode
    elseif cb_opcodes[line] then
        return 0xcb, cb_opcodes[line]
    end

    -- try indexing after stripping each successive word from input.
    -- for example, 'ld (de),16' -> ' (de),16', 'ld (),16', 'ld (de),'.
    local index = 1
    for word in string.gmatch(line, '[%w_.]+') do
        index = string.find(line, word, index)

        local stripped_line = (
            string.sub(line, 1, index - 1) .. string.sub(line, index + #word))

        local op = ops[stripped_line]
        if op and index == op.pindex then
            return compile_op_with_arg(line, op, word, offset, labels, defs)
        end

        index = index + 1 -- don't match the same location twice
    end

    error(line .. ': unknown operation')
end

-- as `compile()`, but treats the input as a single instruction.
local function compile_line(line, offset, labels, defs)
    return string.char(compile_line_to_bytes(line, offset, labels, defs))
end

-- Parses a series of instructions and returns the equivalent machine code as a
-- byte string. Generates an error if an instruction does not match any
-- mnemonic, or if an invalid argument is given to an instruction. The optional
-- `opts` table can have the fields:
--
-- - `delims`, a list (string) of characters that separate instructions in the
--   input (default `'\n'`).
-- - `defs`, a table of string -> number mappings as if constructed by a series
--   of `define` commands. Additional `define`s in the input block will add to
--   the table.
function M.compile(block, opts)
    opts = opts or {}
    local delims = opts.delims or '\n'
    local delim_pattern = string.format('[^%s]+', delims)
    local label_pattern = '^%.[%w_]+$'

    -- first pass: determine label offsets
    local labels = {}
    local offset = 0
    for line in string.gmatch(block, delim_pattern) do
        line = strip_line(line)
        if #line > 0 then
            if string.match(line, label_pattern) then
                if labels[line] then
                    error('duplicate label: ' .. line)
                end
                labels[line] = offset
            else
                offset = offset + #compile_line(line)
            end
        end
    end

    -- second pass:
    local defs = opts.defs or {}
    local instructions = {}
    local offset = 0
    for line in string.gmatch(block, delim_pattern) do
        line = strip_line(line)
        if #line > 0 then
            if not string.match(line, label_pattern) then
                local bytes = compile_line(line, offset, labels, defs)
                table.insert(instructions, bytes)
                offset = offset + #bytes
            end
        end
    end

    return table.concat(instructions)
end

-- reads a single instruction from a block of machine code at the given index.
-- returns the instruction as a mnemonic, and the new index. if `labels` is
-- given, matching labels are used for jr instructions.
local function read_instruction(block, i, labels, defs)
    local opcode = string.byte(block, i)

    if opcode == 0xcb then
        if i + 1 > #block then
            error('missing data after prefix cb')
        end

        return cb_mnemonics[string.byte(block, i + 1)], i + 2
    else
        local mnemonic = mnemonics[opcode]
        if not mnemonic then
            error(string.format('invalid opcode: %02x', opcode))
        end

        if string.find(mnemonic, '%a8') then
            if i + 1 > #block then
                error(string.format(
                    'missing data after opcode %02x', opcode))
            end

            local arg = string.byte(block, i + 1)
            local ins = string.gsub(mnemonic, '%a8',
                string.format('%02x', arg))

            -- substitute labels for jr if possible
            if labels and string.match(mnemonic, '^jr') then
                local jr_dest = arg + i + 1
                if arg >= 0x80 then
                    jr_dest = arg + i - 0xff
                end

                local label = labels[jr_dest]
                if label then
                    ins = string.gsub(mnemonic, '%a8', label)
                end
            end

            return ins, i + 2
        elseif string.find(mnemonic, '%a16') then
            if i + 2 > #block then
                error(string.format(
                    'missing data after opcode %02x', opcode))
            end

            local arg1 = string.byte(block, i + 1)
            local arg2 = string.byte(block, i + 2)
            local arg = arg1 + arg2 * 0x100
            local ins = string.gsub(mnemonic, '%a16',
                string.format('%04x', arg))

            -- substitute def for 16-bit values if possible
            if defs and defs[arg] then
                ins = string.gsub(mnemonic, '%a16', defs[arg])
            end

            return ins, i + 3
        else
            return mnemonic, i + 1
        end
    end
end

-- Converts a string of machine code into an asm string. Generates an error if
-- an opcode is invalid, or if not enough bytes remain in the string to satisfy
-- an instruction's argument. The optional `opts` table can have the fields:
--
-- - `delim`, a sequence of characters that separates instructions in the
--   output (default `'\n'`).
-- - `defs`, a table of string -> number mappings as if constructed by a series
--   of `define` commands. 16-bit values that unambiguously match an entry will
--   use the corresponding symbol.
function M.decompile(block, opts)
    opts = opts or {}

    -- first pass: determine label offsets
    local labels = {}
    local i = 1
    while i <= #block do
        local opcode = string.byte(block, i)
        if not mnemonics[opcode] then
            error(string.format('invalid opcode: %02x', opcode))
        end

        if string.match(mnemonics[opcode], '^jr ') then
            local arg = string.byte(block, i + 1)

            if arg < 0x80 then
                labels[arg + i + 1] = '.next'
            else
                labels[arg + i - 0xff] = '.loop'
            end
        end

        _, i = read_instruction(block, i)
    end

    -- number labels
    local prefix_counts = {}
    for i = 0, #block - 1 do
        local label = labels[i]
        if label then
            if not prefix_counts[label] then
                prefix_counts[label] = 0
            else
                labels[i] = string.format(
                    '%s%d', label, prefix_counts[label] + 1)
            end

            prefix_counts[label] = prefix_counts[label] + 1
        end
    end

    -- make reverse lookup table for defines
    local duplicate_values = {}
    local reverse_defs = {}
    for symbol, value in pairs(opts.defs or {}) do
        if not duplicate_values[value] then
            if reverse_defs[value] then
                duplicate_values[value] = true
                reverse_defs[value] = nil
            else
                reverse_defs[value] = symbol
            end
        end
    end

    -- second pass:
    local lines = {}
    local i = 1
    while i <= #block do
        if labels[i - 1] then
            table.insert(lines, labels[i - 1])
        end

        local ins = nil
        ins, i = read_instruction(block, i, labels, reverse_defs)
        table.insert(lines, ins)
    end

    return table.concat(lines, opts.delim or '\n')
end

return M
