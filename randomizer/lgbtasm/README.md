# Lua Game Boy: The Assembler

## Summary

The good:

- Converts assembly to machine code
- Converts machine code to assembly
- No external dependencies
- Lua 5.1 compatible

The bad:

- Not very tested
- Very limited command language
- It's only a module and nothing executable


## Usage

This module uses bgb/no$gmb syntax and enforces a strict style: numbers are
undecorated (no `$`, etc.) and hexadecimal, `a,` is required in mnemonics
that feature it, spaces do not appear after `,`s, parens are used for
"dereferencing" memory, and all keywords are lower-case. User-defined
symbols such as labels are case-sensitive. A label and instruction cannot
appear on the same line.

The characters in `;*#` all begin inline comments, although instruction
delimiter status overrides comment character status in the `compile()`
function.

"Local" labels (the only kind) start with a `.` and can be referenced by
relative jumps. Decompilation automatically generates labels for relative
jump destinations.


## Functions

### `compile(block, opts)`

Parses a series of instructions and returns the equivalent machine code as a
byte string. Generates an error if an instruction does not match any
mnemonic, or if an invalid argument is given to an instruction. The optional
`opts` table can have the fields:

- `delims`, a list (string) of characters that separate instructions in the
  input (default `'\n'`).
- `defs`, a table of string -> number mappings as if constructed by a series
  of `define` commands. Additional `define`s in the input block will add to
  the table.

```
> lgbtasm = require 'lgbtasm'
> lgbtasm.compile('cp b; ret', {delims = ';'}) == '\xb8\xc9'
true
> lgbtasm.compile('ld (x),a', {defs = {x = 0xc6c5}}) == '\xea\xc5\xc6'
true
```

### `decompile(block, opts)`

Converts a string of machine code into an asm string. Generates an error if
an opcode is invalid, or if not enough bytes remain in the string to satisfy
an instruction's argument. The optional `opts` table can have the fields:

- `delim`, a sequence of characters that separates instructions in the
  output (default `'\n'`).
- `defs`, a table of string -> number mappings as if constructed by a series
  of `define` commands. 16-bit values that unambiguously match an entry will
  use the corresponding symbol.

```
> lgbtasm = require 'lgbtasm'
> lgbtasm.decompile('\xb8\xc9', {delim = '; '})
cp b; ret
> lgbtasm.decompile('\xfa\xc5\xc6', {defs = {x = 0xc6c5}})
ld a,(x)
```


## Commands

The following syntax is represented in bastardized EBNF.

### `db d8{,d8}`

Creates a sequence of literal bytes.

```
> lgbtasm = require 'lgbtasm'
> lgbtasm.compile('db 01,02,03') == '\x01\x02\x03'
true
```

### `dw d16{,d16}`

Creates a sequence of big-endian words.

```
> lgbtasm = require 'lgbtasm'
> lgbtasm.compile('dw 0201,0403') == '\x01\x02\x03\x04'
true
```

### `define symbol,value`

Associates a symbol with a constant numeric value. Redefining a symbol is not
an error.

```
> lgbtasm = require 'lgbtasm'
> lgbtasm.compile('define x,01; ld a,x', {delims = ';'}) == '\x3e\x01'
true
```
