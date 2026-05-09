# Tya v0.27 Specification

This document is the specification for Tya v0.27 after v0.26 external
package management.

## Theme

Tya v0.27 adds hexadecimal and binary integer literals.

These are small additions to the lexer and number-literal handling. They do
not change the value model: every numeric literal still produces an `int`
or `float` of the same kind as before. Binary and hex literals are simply a
more readable way to write integer constants — particularly useful for
bit-level work introduced in v0.25 (NES emulators, network protocols,
CPU-flag tables).

## Goals

- Add hexadecimal integer literals (`0xFF`).
- Add binary integer literals (`0b1010`).
- Allow underscores between digits as visual separators.
- Keep all other syntax, types, and semantics unchanged.

## Included in v0.27

v0.27 includes all v0.26 behavior and adds:

- Hexadecimal integer literals
- Binary integer literals
- Digit-group underscore separators (`1_000_000`, `0xff_ee`, `0b1010_0011`)

## Not Included in v0.27

v0.27 does not include:

- Octal literals (`0o755`)
- Hexadecimal float literals (`0x1.fp10`)
- Numeric type suffixes (`123u`, `1.5f`)
- Arbitrary-precision (big-int / big-decimal)
- Implicit base inference for parsed strings

## Hexadecimal Literals

A hexadecimal integer literal starts with `0x` or `0X` followed by one or
more hexadecimal digits (`0`-`9`, `a`-`f`, `A`-`F`).

```tya
255           # decimal
0xff          # 255
0xFF          # 255
0xCAFE        # 51966
0x00          # 0
```

Lower-case and upper-case digits both work. The `0x` / `0X` prefix is also
case-insensitive.

Hex literals are always integers (`int`). They have no fractional form in
v0.27.

Underscores may appear between hex digits as visual separators:

```tya
0xff_ee_dd
0x_dead_beef
```

A leading underscore directly after `0x` is permitted; trailing underscores
and consecutive underscores within the digit run are also permitted as long
as at least one hex digit appears.

## Binary Literals

A binary integer literal starts with `0b` or `0B` followed by one or more
binary digits (`0`, `1`).

```tya
0b1010        # 10
0b1111_1111   # 255
0b0           # 0
```

Underscore separators apply the same way as for hex literals.

## Decimal Literal Underscores

Plain decimal literals also accept underscores between digits:

```tya
1_000
1_000_000
3.14
1_000.5
```

Underscores are illegal at the start of a decimal literal (`_123` is an
identifier, not a number) and inside the fractional part of a float they
are also allowed (`1_000.500_500`).

## Negative Literals

There is no negative-prefix form for integer literals. `-0xff` parses as
the unary `-` operator applied to `0xff`, identical to the existing
behavior for `-1`.

```tya
x = -0xff      # -255
y = -0b10      # -2
```

## Equivalence

Hex, binary, and decimal literals produce **identical** Tya values when
they represent the same integer. The compiler and runtime see no
distinction after parsing:

```tya
255 == 0xff           # true
255 == 0b1111_1111    # true
kind(0xff)            # "int"
```

## Diagnostics

v0.27 implementations should report source-oriented errors for:

- a `0x` prefix not followed by at least one hex digit
- a `0b` prefix not followed by at least one binary digit
- a hex literal containing a non-hex digit (e.g. `0x1g`)
- a binary literal containing a digit other than `0` or `1`
- two adjacent underscores at the start of a literal where no digit has
  yet appeared

Diagnostics should mention the offending literal and indicate whether the
prefix is `0x` or `0b`.

## Implementation Notes (non-normative)

- Lexer: when the digit-scan starts and the input is `0x`/`0X` or `0b`/
  `0B`, switch to a hex- or binary-digit run, accept underscores, and emit
  an `INT` token with the canonical decimal lexeme so the existing
  `strconv.ParseInt(text, 10, 64)` path continues to work. Alternatively,
  pass the value through directly as `strconv.ParseInt(_, 16, 64)` and
  store the canonical decimal text in the token lexeme.
- Codegen: nothing changes; the token already carries an integer value.
- Eval: nothing changes; literal evaluation reads the lexeme.

These notes are guidance, not the spec; conforming implementations may
differ as long as the user-visible behavior matches.
