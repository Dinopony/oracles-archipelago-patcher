#!/usr/bin/env lua

lgbtasm = require '.../lgbtasm'


-- compiling errors:

-- unknown operation
status, err = pcall(lgbtasm.compile, 'unknown')
assert(status == false and string.match(err, 'unknown operation'))

-- define not found
status, err = pcall(lgbtasm.compile, '.next\nld a,.next')
assert(status == false and string.match(err, 'define not found'))

-- label not found
status, err = pcall(lgbtasm.compile, 'jr x', {defs = {x = 1}})
assert(status == false and string.match(err, 'local label not found'))

-- 16-bit arg for 8-bit instruction
status, err = pcall(lgbtasm.compile, 'ld a,c692')
assert(status == false and string.match(err, 'invalid argument'))

-- duplicate labels
status, err = pcall(lgbtasm.compile, '.next\njr .next\n.next')
assert(status == false and string.match(err, 'duplicate label'))


-- compiling valid asm:

-- nullary instruction
s = lgbtasm.compile('ld b,a')
assert(s == '\x47')

-- unary instruction
s = lgbtasm.compile('ld a,3f')
assert(s == '\x3e\x3f')

-- binary instruction
s = lgbtasm.compile('ld hl,c692')
assert(s == '\x21\x92\xc6')

-- instruction involving an address
s = lgbtasm.compile('ld (cc49),a')
assert(s == '\xea\x49\xcc')

-- prefix cb instruction
s = lgbtasm.compile('bit 4,a')
assert(s == '\xcb\x67')

-- ldh-type instruction
s = lgbtasm.compile('ld (ff00+b5),a')
assert(s == '\xe0\xb5')

-- instruction using define
s = lgbtasm.compile('ld a,x', {defs = {x = 0x3f}})
assert(s == '\x3e\x3f')

-- line with indent and comment
s = lgbtasm.compile('    cp b ; comment')
assert(s == '\xb8')

-- multiple instructions, newline-delimited
s = lgbtasm.compile('cp b\nret')
assert(s == '\xb8\xc9')

-- multiple instructions, semicolon-delimited
s = lgbtasm.compile('cp b; ret', {delims = ';'})
assert(s == '\xb8\xc9')

-- block with commented blank line
s = lgbtasm.compile('cp b\n; comment\nret')
assert(s == '\xb8\xc9')

-- block with invalid instruction
status = pcall(lgbtasm.compile, 'cp b\ninvalid\nret')
assert(status == false)

-- forward jump to label
s = lgbtasm.compile('jr .next\ncp a,49\n.next')
assert(s == '\x18\x02\xfe\x49')

-- backward jump to label
s = lgbtasm.compile('.loop\ncp a,49\njr .loop')
assert(s == '\xfe\x49\x18\xfc')

-- multiple labels
s = lgbtasm.compile([[.loop
ld a,a
jr .next
.loop2
ld a,b
jr .next2
ld a,c
jr .next
.next
ld a,d
jr .loop2
.next2
ld a,e
jr .loop]])
assert(s == '\x7f\x18\x06\x78\x18\x06\x79\x18\x00\x7a\x18\xf7\x7b\x18\xf1')


-- compiling assembler commands:

-- db with no arguments
status, err = pcall(lgbtasm.compile, 'db')
assert(status == false and string.match(err, 'missing argument'))

-- single-entry db
s = lgbtasm.compile('db 1a')
assert(s == '\x1a')

-- multiple-entry db
s = lgbtasm.compile('db 1a,2b,3c')
assert(s == '\x1a\x2b\x3c')

-- db with defines
s = lgbtasm.compile('db x,02,z', {defs = {x = 1, z = 3}})
assert(s == '\x01\x02\x03')

-- dw without defines
s = lgbtasm.compile('dw 0201,0403,0605')
assert(s == '\x01\x02\x03\x04\x05\x06')

-- dw with defines
s = lgbtasm.compile('dw x,0002,z', {defs = {x = 1, z = 3}})
assert(s == '\x01\x00\x02\x00\x03\x00')

-- db overflow
status, err = pcall(lgbtasm.compile, 'db 0201,0403')
assert(status == false and string.match(err, 'invalid argument'))

-- define with insufficient arguments
status, err = pcall(lgbtasm.compile, 'define x')
assert(status == false and string.match(err, 'invalid define'))

-- define with bad name
status, err = pcall(lgbtasm.compile, 'define 1,01')
assert(status == false and string.match(err, 'invalid define'))

-- define with bad value
status, err = pcall(lgbtasm.compile, 'define x,y')
assert(status == false and string.match(err, 'invalid define'))

-- valid define
defs = {}
s = lgbtasm.compile('define x,01', {defs = defs})
assert(#s == 0 and defs.x == 0x01)

-- valid define in context
s = lgbtasm.compile('define x,01\nld a,x')
assert(s == '\x3e\x01' and defs.x == 0x01)


-- decompiling errors:

-- decompile invalid opcode
status, err = pcall(lgbtasm.decompile, '\xd3')
assert(status == false and string.match(err, 'invalid opcode'))

-- decompile unary instruction w/o enough data for arg
status, err = pcall(lgbtasm.decompile, '\x3e')
assert(status == false and string.match(err, 'missing data'))

-- decompile binary instruction w/o enough data for arg
status, err = pcall(lgbtasm.decompile, '\x21\x92')
assert(status == false and string.match(err, 'missing data'))

-- decompile prefix cb instruction w/o enough data for arg
status, err = pcall(lgbtasm.decompile, '\xcb')
assert(status == false and string.match(err, 'missing data'))


-- decompiling valid code:

-- decompile nullary instruction
s = lgbtasm.decompile('\xaf')
assert(s == 'xor a')

-- decompile unary instruction
s = lgbtasm.decompile('\x3e\x3f')
assert(s == 'ld a,3f')

-- decompile binary instruction
s = lgbtasm.decompile('\x21\x92\xc6')
assert(s == 'ld hl,c692')

-- decompile prefix cb instruction
s = lgbtasm.decompile('\xcb\x67')
assert(s == 'bit 4,a')

-- decompile ldh-type instruction
s = lgbtasm.decompile('\xe0\xb5')
assert(s == 'ld (ff00+b5),a')

-- decompile block
s = lgbtasm.decompile('\x3e\x3f\xcb\x67\xc9')
assert(s == 'ld a,3f\nbit 4,a\nret')

-- decompile block with semicolon delimiter
s = lgbtasm.decompile('\x3e\x3f\xcb\x67\xc9', {delim = '; '})
assert(s == 'ld a,3f; bit 4,a; ret')

-- decompile forward jump to label
s = lgbtasm.decompile('\x18\x02\xfe\x49\xc9')
assert(s == 'jr .next\ncp a,49\n.next\nret')

-- decompile backward jump to label
s = lgbtasm.decompile('\xfe\x49\x18\xfc')
assert(s == '.loop\ncp a,49\njr .loop')

-- decompile multiple labels
s = lgbtasm.decompile(
    '\x7f\x18\x06\x78\x18\x06\x79\x18\x00\x7a\x18\xf7\x7b\x18\xf1')
assert(s == [[.loop
ld a,a
jr .next
.loop2
ld a,b
jr .next2
ld a,c
jr .next
.next
ld a,d
jr .loop2
.next2
ld a,e
jr .loop]])

-- decompile with defs
s = lgbtasm.decompile(
    '\x3e\x01\x21\x01\x00\x21\x02\x00', {defs = {x = 1, y = 2, z = 2}})
assert(s == 'ld a,01\nld hl,x\nld hl,0002')
