# Tya v0.59 Specification

> **Status:** shipped. The `tya version` constant is `0.59.0`.
> v0.59 makes every primitive literal a class instance:
> `42.to_s()`, `"hi".upper()`, `[1,2].len()` all dispatch through
> the wrapper class. The `kind(x)` builtin is removed in favor of
> `x.class` / `x.class.name`. The module-style `string.upper(s)`,
> `array.len(a)`, `dict.keys(d)` etc. are removed in favor of the
> method-style. No new keywords are added.

## Theme

v0.5 introduced `class`. v0.44 made everything a class
(class-oriented namespace). v0.59 extends "everything is a class"
to the last remaining holdouts — the literal values themselves.

After v0.59, `42` is a `Number` instance, `"hi"` is a `String`
instance, `[1, 2]` is an `Array` instance, `{a: 1}` is a `Dict`
instance, `true` / `false` are `Boolean` instances, and `nil` is
the unique `Nil` instance. Method calls on literals work
naturally (`42.to_s()`, `"hi".len()`), operators desugar to
double-underscore method names (`a + b` → `a.__add__(b)`), and
type introspection happens through `x.class`.

The runtime representation is unchanged — `TyaValue` stays a
tagged union (`runtime/tya_runtime.h`), `kind` enum and payload
fields are the same, no boxing of primitives. The wrapper classes
are process-global singletons created once at startup; `x.class`
returns the appropriate singleton with no allocation.

## The six primitive classes

| Literal form | Class | Runtime `kind` |
|---|---|---|
| `42`, `1.0`, `-3.14`, `0x2a`, `0b101010`, `1e9` | `Number` | `TYA_NUMBER` |
| `"hi"`, `r"raw"`, `"interp {x}"` | `String` | `TYA_STRING` |
| `[1, 2, 3]`, `[]` | `Array` | `TYA_ARRAY` |
| `{a: 1}`, `{}` | `Dict` | `TYA_DICT` |
| `true`, `false` | `Boolean` | `TYA_BOOL` |
| `nil` | `Nil` | `TYA_NIL` |

The six class identifiers (`Number`, `String`, `Array`, `Dict`,
`Boolean`, `Nil`) are **reserved** at the top level. User code
cannot rebind them.

Integer and float are not distinguished — `1` and `1.0` are both
`Number` instances and `1 == 1.0` is `true`. The runtime stores
all numeric values as `double`. A future Epic may introduce a
separate big-integer or fixed-precision class; the promotion
rules between them will be defined at that time.

Byte-sequence literals (`b"..."`, v0.25) remain a separate value
kind backed by `TYA_BYTES`. v0.59 does **not** introduce a
`Bytes` primitive class — bytes are still handled by the `bytes`
module surface and by indexing. A future Epic may add a `Bytes`
wrapper class on the same pattern as the six classes here.

The set literal `{1, 2, 3}` and a `Set` class are out of scope.
For v0.59 the curly-brace literal continues to mean "dict only"
(`{}` is the empty dict, `{a: 1}` is a single-entry dict; an
unkeyed `{1, 2, 3}` is a syntax error).

## `x.class`

`x.class` returns the wrapper class for `x` as a process-global
singleton. No allocation, no method dispatch — the C emitter
inlines it to a static lookup keyed on `x.kind`.

```tya
42.class             # → Number
"hi".class           # → String
[1, 2].class         # → Array
{a: 1}.class         # → Dict
true.class           # → Boolean
nil.class            # → Nil

42.class == Number       # → true (identity comparison)
42.class == "Number"     # → false (Class is not a String)
42.class.name            # → "Number"  (String)
```

Six class identifiers (`Number`, `String`, `Array`, `Dict`,
`Boolean`, `Nil`) at the top level evaluate to the corresponding
singleton, so `42.class == Number` is the idiomatic type check.

For user-defined classes the same applies:

```tya
class Point
  initialize = x, y ->
    self.x = x
    self.y = y

p = Point(1, 2)
p.class             # → Point (the class object)
p.class.name        # → "Point"
p.class == Point    # → true
```

`x.class.name` returns a `String` whose value is the class name.
For primitive classes this is exactly one of `"Number"`,
`"String"`, `"Array"`, `"Dict"`, `"Boolean"`, `"Nil"`. For user
classes it is the declared name.

`Number.class` and the analogous queries on the wrapper classes
themselves are deliberately undefined for v0.59 — a `Class` /
metaclass tower is out of scope. Implementations may either
return a placeholder, raise, or refuse at compile time; tooling
should not rely on the result.

## Method-call syntax on literals

```tya
42.to_s()                # "42"
1.5.floor()              # 1
"hi".upper()             # "HI"
"hi,there".split(",")    # ["hi", "there"]
[1, 2, 3].len()          # 3
[1, 2, 3].push(4)        # [1, 2, 3, 4]
{a: 1, b: 2}.keys()      # ["a", "b"]
true.to_s()              # "true"
nil.to_s()               # "nil"
```

Method-call syntax was already available for variables in v0.16+
(via dynamic `tya_member` dispatch); v0.59 makes it work on raw
literals as well by giving each literal a fixed wrapper class.

### Lexer disambiguation: `42.0` vs `42.foo`

The lexer treats a `.` that immediately follows a numeric literal
as either the decimal point of a float literal or as the
member-access dot, based on the next character:

| Source | Tokens | Notes |
|---|---|---|
| `42` | `NUMBER(42)` | integer-valued Number |
| `42.0` | `NUMBER(42.0)` | float-valued Number |
| `42.5e2` | `NUMBER(4250.0)` | scientific notation |
| `42.foo()` | `NUMBER(42) DOT IDENT(foo) ...` | method call on `42` |
| `42.to_s()` | `NUMBER(42) DOT IDENT(to_s) ...` | method call on `42` |
| `42.` (EOF) | error | trailing `.` without digit or ident |
| `42 .foo()` | `NUMBER(42) DOT IDENT(foo) ...` | whitespace makes the dot a method dot |

The rule is **Ruby's**: a digit (`0-9`) immediately after `.`
makes the dot part of the float literal; an alphabetic character
or `_` makes it a method-access dot. The current lexer already
implements this rule (since pre-v0.16); v0.59 just locks it in.

## Operator desugaring

Every built-in operator desugars to a fixed double-underscore
method name on the receiver class. Operators are **not
user-redefinable** on the built-in primitive classes
(monkey-patching is forbidden — see *Rules* below); user classes
may *define* these methods to participate in operator syntax.

| Operator | Arity | Desugars to | Notes |
|---|---|---|---|
| `a + b` | binary | `a.__add__(b)` | Number: add; String/Array: concat |
| `a - b` | binary | `a.__sub__(b)` | Number only on built-ins |
| `a * b` | binary | `a.__mul__(b)` | Number only on built-ins |
| `a / b` | binary | `a.__div__(b)` | Number only on built-ins; IEEE 754 |
| `a % b` | binary | `a.__mod__(b)` | Number only on built-ins; `fmod` |
| `-a` | unary | `a.__neg__()` | Number only on built-ins |
| `a == b` | binary | `a.__eq__(b)` | type-strict on built-ins |
| `a != b` | binary | `!(a.__eq__(b))` | derived |
| `a < b` | binary | `a.__lt__(b)` | Number and String |
| `a <= b` | binary | `a.__le__(b)` | derived from `__lt__` and `__eq__` |
| `a > b` | binary | `b.__lt__(a)` | derived |
| `a >= b` | binary | `b.__le__(a)` | derived |
| `a & b` | binary | `a.__bitand__(b)` | Number (truncated to int64) |
| `a \| b` | binary | `a.__bitor__(b)` | Number (truncated to int64) |
| `a ^ b` | binary | `a.__bitxor__(b)` | Number (truncated to int64) |
| `a << b` | binary | `a.__shl__(b)` | Number (truncated to int64) |
| `a >> b` | binary | `a.__shr__(b)` | Number (truncated to int64) |
| `~a` | unary | `a.__bitnot__()` | Number (truncated to int64) |
| `a[k]` | binary | `a.__index__(k)` | Array/Dict/String read |
| `a[k] = v` | ternary | `a.__index_set__(k, v)` | Array/Dict write |

Short-circuiting logical operators (`&&`, `\|\|`, `!`) are
**not** desugared. They retain their evaluation semantics
(`a && b` does not evaluate `b` if `a` is falsy) and dispatch
through truthiness, not through a method on the receiver. There
is therefore no `__and__` / `__or__` / `__not__` on the wrapper
classes for these.

## Method tables for the six classes

The following lists are exhaustive — these are the methods
guaranteed to exist on every instance of each class. The actual
implementation comes from the existing builtins / runtime
helpers; v0.59 changes the **surface**, not the semantics.

### `Number`

```
to_s() -> String                      "42", "1.5"
to_i() -> Number                      truncate to integer-valued Number
to_f() -> Number                      identity (kept for symmetry / clarity)
abs() -> Number
floor() -> Number
ceil() -> Number
round() -> Number
trunc() -> Number
sqrt() -> Number
pow(other: Number) -> Number
log() -> Number                       natural log
log2() -> Number
log10() -> Number
exp() -> Number
sin() / cos() / tan() -> Number
asin() / acos() / atan() -> Number
atan2(other: Number) -> Number
integer?() -> Boolean                 true when value has no fractional part
finite?() -> Boolean
nan?() -> Boolean

__add__(b) __sub__(b) __mul__(b) __div__(b) __mod__(b)
__neg__()
__eq__(b) __lt__(b) __le__(b)
__bitand__(b) __bitor__(b) __bitxor__(b) __shl__(b) __shr__(b) __bitnot__()
class -> Class
```

### `String`

```
len() -> Number                       character count (UTF-8 aware)
byte_len() -> Number                  raw byte length
char_len() -> Number                  alias of len()
upper() -> String
lower() -> String
trim() -> String
contains(needle: String) -> Boolean
starts_with(prefix: String) -> Boolean
ends_with(suffix: String) -> Boolean
replace(old: String, new: String) -> String
split(sep: String) -> Array
chars() -> Array                      array of single-char Strings
bytes() -> Bytes                      UTF-8 byte representation
to_s() -> String                      identity
to_i() -> Number                      parse; raises on garbage
to_f() -> Number                      parse; raises on garbage
to_number() -> Number                 parse; raises on garbage
blank?() -> Boolean                   trim() == ""
present?() -> Boolean                 trim() != ""

__add__(b: String) -> String          concatenation
__eq__(b) -> Boolean                  byte-equal, type-strict
__lt__(b: String) -> Boolean          lexicographic by bytes
__le__(b: String) -> Boolean
__index__(i: Number) -> String        single-char substring; raises on OOB
class -> Class
```

`split("")` returns the character array (same as `chars()`).

### `Array`

```
len() -> Number
empty?() -> Boolean
first() -> any                        nil when empty
last() -> any                         nil when empty
push(v: any) -> Array                 returns self, mutates
pop() -> any                          removes and returns last
join(sep: String) -> String
map(fn) -> Array
filter(fn) -> Array
find(fn) -> any                       nil when no match
any(fn) -> Boolean
all(fn) -> Boolean
reduce(initial, fn) -> any
contains(v: any) -> Boolean
slice(start: Number, end: Number) -> Array
reverse() -> Array                    new array
sort() -> Array                       new array, ascending
sort_by(fn) -> Array                  new array
to_s() -> String                      "[1, 2, 3]"

__add__(b: Array) -> Array            concatenation, new array
__eq__(b) -> Boolean                  element-wise equal, type-strict
__index__(i: Number) -> any           raises on OOB
__index_set__(i: Number, v: any)
class -> Class
```

### `Dict`

```
len() -> Number
empty?() -> Boolean
has(k: String) -> Boolean
get(k: String) -> any                 nil when missing
get(k: String, default: any) -> any
set(k: String, v: any) -> Dict        returns self, mutates
delete(k: String) -> any              returns removed value or nil
keys() -> Array
values() -> Array
entries() -> Array                    array of [k, v] pairs
merge(other: Dict) -> Dict            new dict; later wins
to_s() -> String                      "{a: 1, b: 2}"

__eq__(b) -> Boolean                  same keys + equal values, type-strict
__index__(k: String) -> any           nil when missing
__index_set__(k: String, v: any)
class -> Class
```

### `Boolean`

```
to_s() -> String                      "true" or "false"
__eq__(b) -> Boolean
class -> Class
```

### `Nil`

```
to_s() -> String                      "nil"
__eq__(b) -> Boolean                  true only for nil itself
class -> Class
```

## `kind` removal

The `kind(x)` builtin is **removed**. v0.58 returned one of the
strings `"nil"`, `"bool"`, `"int"`, `"float"`, `"string"`,
`"array"`, `"dict"`; v0.59 has no equivalent string builtin. Use
`x.class` (class identity comparison) or `x.class.name`
(string).

### Migration

| v0.58 | v0.59 |
|---|---|
| `kind(x) == "int"` or `kind(x) == "float"` | `x.class == Number` |
| `kind(x) == "string"` | `x.class == String` |
| `kind(x) == "array"` | `x.class == Array` |
| `kind(x) == "dict"` | `x.class == Dict` |
| `kind(x) == "bool"` | `x.class == Boolean` |
| `kind(x) == "nil"` | `x.class == Nil` |
| Need a string label? | `x.class.name` |

Calling `kind(...)` in v0.59 raises `TYA-E0810 kind builtin
removed in v0.59; use x.class or x.class.name`.

## stdlib API consolidation

The module-style facade over the same operations is **removed**.

| v0.58 (`module X` function-style) | v0.59 (method on the wrapper class) |
|---|---|
| `string.len(s)` | `s.len()` |
| `string.trim(s)` | `s.trim()` |
| `string.contains(s, n)` | `s.contains(n)` |
| `string.starts_with(s, p)` | `s.starts_with(p)` |
| `string.ends_with(s, p)` | `s.ends_with(p)` |
| `string.replace(s, old, new)` | `s.replace(old, new)` |
| `string.split(s, sep)` | `s.split(sep)` |
| `string.join(values, sep)` | `values.join(sep)` |
| `string.blank(s)` | `s.blank?()` |
| `string.present(s)` | `s.present?()` |
| `array.len(a)` | `a.len()` |
| `array.empty(a)` | `a.empty?()` |
| `array.first(a)` | `a.first()` |
| `array.pop(a)` | `a.pop()` |
| `array.join(a, sep)` | `a.join(sep)` |
| `array.map(a, fn)` | `a.map(fn)` |
| `array.filter(a, fn)` | `a.filter(fn)` |
| `array.find(a, fn)` | `a.find(fn)` |
| `array.any(a, fn)` | `a.any(fn)` |
| `array.all(a, fn)` | `a.all(fn)` |
| `array.reduce(a, init, fn)` | `a.reduce(init, fn)` |
| `dict.len(d)` | `d.len()` |
| `dict.has(d, k)` | `d.has(k)` |
| `dict.keys(d)` | `d.keys()` |
| `dict.values(d)` | `d.values()` |

The free-function builtins that did the underlying work
(`len(...)`, `trim(...)`, `contains(...)`, `keys(...)`,
`values(...)`, `has(...)`, `push(...)`, `pop(...)`,
`map(...)`, `filter(...)`, `find(...)`, `any(...)`, `all(...)`,
`reduce(...)`, `join(...)`, `split(...)`, `replace(...)`,
`starts_with(...)`, `ends_with(...)`) are **also removed** at
the top level. Their behaviour is reachable only as methods on
the wrapper classes. `to_s(x)` / `to_string(x)` /
`to_int(x)` / `to_float(x)` / `to_number(x)` survive *only* as
methods (`x.to_s()`, `x.to_i()`, etc.) — the top-level builtin
forms are removed.

`print` and `println` remain as top-level builtins (they are
not methods on a class, by design; the receiver is implicit).
`args`, `exit`, `panic`, `assert`, `assert_equal`, `equal`,
`error`, `chr`, `ord` remain as top-level builtins.

`stdlib/string.tya`, `stdlib/array.tya`, `stdlib/dict.tya`
are **deleted**. Any program importing them via
`import string` / `import array` / `import dict` raises a
load-time error `TYA-E0811 module string|array|dict was
removed in v0.59; methods now live on the wrapper class`.

The module-style `math.*` (`math.sqrt`, `math.floor`, ...)
remains but the same methods are also exposed on `Number`. Both
are first-class. Module-style is retained because `math` is
operated on by the **second** argument (`math.atan2(y, x)`) and
the natural-language form `y.atan2(x)` reads less obviously than
`math.atan2(y, x)` for some callers — both surfaces are kept
for v0.59. Future Epics may revisit. Modules unrelated to the
six primitive classes (`time`, `random`, `digest`, `file`, `os`,
`path`, `process`, `json`, `toml`, `csv`, `base64`, `hex`,
`url`, `secure_random`, `matrix`, `markdown`, `net`, `channel`,
`sync`, `task`, `runtime`, `value`, `unittest`) are unaffected.

## Subclassing rule

The six built-in primitive classes are **final**. Declaring a
class that inherits from any of them is a compile-time error.

```tya
class MyNumber < Number     # ✗ TYA-EXXXX
class MyString < String     # ✗ TYA-EXXXX
class MyArray  < Array      # ✗ TYA-EXXXX
class MyDict   < Dict       # ✗ TYA-EXXXX
class MyBool   < Boolean    # ✗ TYA-EXXXX
class MyNil    < Nil        # ✗ TYA-EXXXX
```

Reasons: the optimizer relies on the wrapper classes' method
tables being fixed at compile time so that the fast path
(operator desugaring lowered directly to C helpers, no method
dispatch) is unconditional. Subclassing would require a
"redefinition check" similar to CRuby's redefined-method flags,
which we are choosing to avoid.

User code that wants Number-like behaviour declares its own
class and defines the operator methods (`__add__`, `__eq__`,
etc.); operator syntax then dispatches normally.

## Monkey-patching rule

The method tables of the six built-in primitive classes are
**fixed at compile time**. Adding, replacing, or removing a
method on `Number` / `String` / `Array` / `Dict` / `Boolean` /
`Nil` is a compile-time error.

```tya
Number.banana = -> "yellow"    # ✗ TYA-EXXXX
String.upper = -> "no"         # ✗ TYA-EXXXX (redefine)
```

The same restriction applies to operator methods (`__add__`,
`__eq__`, etc.) on the built-in classes. User-defined classes
have no such restriction — they can be re-opened and extended
through the normal `class` declaration shape.

## Cross-type equality

Equality is **method-level**. The built-in `__eq__`
implementations are **type-strict**:

```tya
"42" == 42                # → false   (String#__eq__ rejects non-String)
[1] == "1"                # → false   (Array#__eq__ rejects non-Array)
nil == false              # → false   (Nil#__eq__ accepts only nil)
1 == 1.0                  # → true    (both are Number)
```

User classes are free to implement lenient comparison by
returning `true` for cross-type cases inside their own
`__eq__`. The built-in classes will never silently coerce.

Ordering (`<`, `<=`, `>`, `>=`) on cross-type operands is a
type error. The built-in `__lt__` on `Number` and `String`
rejects non-matching argument types with
`TYA-EXXXX type error: cannot compare Number with String`.

## Runtime representation

Unchanged from v0.58:

- `TyaValue` is the same tagged union (`runtime/tya_runtime.h`):
  `kind` enum + payload union. `TYA_NUMBER` stores `double`,
  `TYA_STRING` stores `const char *`, etc.
- No boxing of primitives into heap objects. `42` is the same
  in-memory value as it was in v0.58.
- The six wrapper classes are process-global singleton class
  objects, allocated once during runtime startup. They live for
  the entire process lifetime.
- `x.class` reads `x.kind` and returns the matching singleton.
  Zero allocation, no method dispatch.
- The wrapper class's method table is laid out as a flat
  C-level dispatch (a `switch` on method name hash or a small
  perfect-hashed array), built once at startup.

## Hidden fast path

The C emitter applies the following lowering rules:

1. When *both* operands of a binary operator have statically
   known primitive types — for example `Number + Number`,
   `String + String`, comparisons between `Number` and `Number`,
   etc. — the operator desugars directly to the existing
   runtime helper (`tya_add_number`, `tya_concat_string`,
   `tya_lt_number`, ...). No method dispatch, no allocation of
   a method-call frame, no `x.__add__(y)` lookup.

2. When one operand is statically known to be a user class with
   the relevant method — `Foo.__add__` declared in the source —
   the emitter generates a direct call to that method.

3. When the types are dynamic (variable of unknown type), the
   emitter generates a `tya_dispatch_method(receiver, "__add__",
   args, count)` call. The dispatcher inspects the receiver's
   `kind` field at runtime, looks up the method on the wrapper
   class (or user class), and invokes it.

Because monkey-patching and operator redefinition on built-in
classes are forbidden, the static fast path is **unconditional**
when both operands are known primitives. There is no
redefinition-check flag of the kind CRuby maintains for
optimised arithmetic.

## Diagnostic codes

Newly minted in v0.59:

| Code | Meaning |
|---|---|
| `TYA-E0810` | `kind` builtin removed in v0.59; use `x.class` or `x.class.name` |
| `TYA-E0811` | module `string` / `array` / `dict` was removed in v0.59; methods now live on the wrapper class |
| `TYA-E0812` | top-level builtin `len` / `trim` / `keys` / `push` / ... was removed in v0.59; method now lives on the wrapper class |
| `TYA-E0813` | cannot inherit from built-in primitive class `Number` / `String` / `Array` / `Dict` / `Boolean` / `Nil` |
| `TYA-E0814` | cannot add or redefine method `X` on built-in primitive class `Y` |
| `TYA-E0815` | cannot rebind reserved class identifier `Number` / `String` / `Array` / `Dict` / `Boolean` / `Nil` |
| `TYA-E0816` | type error: cannot compare or operate on `T1` with `T2` |
| `TYA-E0817` | no method `name` on class `C` |

The exact final-digit assignments may shift during
implementation; the table is locked at SPEC-freeze time but the
implementing PRs are authoritative once merged.

## Migration guide

Required edits to upgrade a v0.58 program to v0.59:

| Pattern | Before | After |
|---|---|---|
| Type test | `kind(x) == "int"` | `x.class == Number` |
| Type label | `kind(x)` | `x.class.name` (drop the `"int"`/`"float"` distinction; both yield `"Number"`) |
| String op | `string.upper(s)` | `s.upper()` |
| String op | `string.trim(s)` | `s.trim()` |
| Array op | `array.len(a)` | `a.len()` |
| Array op | `array.map(a, fn)` | `a.map(fn)` |
| Dict op | `dict.keys(d)` | `d.keys()` |
| Builtin | `len(x)` | `x.len()` |
| Builtin | `to_string(x)` | `x.to_s()` |
| Builtin | `to_int(x)` | `x.to_i()` |
| Builtin | `push(a, v)` | `a.push(v)` |
| Builtin | `pop(a)` | `a.pop()` |
| Builtin | `keys(d)` | `d.keys()` |
| Builtin | `map(a, fn)` | `a.map(fn)` |
| Import | `import string` | (delete; methods move to `String`) |
| Import | `import array` | (delete) |
| Import | `import dict` | (delete) |

`tya fmt` does **not** rewrite literals — `42` stays `42` in
source, not `Number(42)`. The class identity is metadata about
the literal, not part of its written form.

`tya lint` will gain a rule (TYAL000X) that detects the v0.58
patterns above and offers `--fix` autocorrect. The lint rule is
informational for one release after v0.59 ships, after which
the diagnostics in the table above (compile-time errors) take
over.

## Scope-out (v0.60+)

- **`Set` literal and `Set` class** — `{1, 2, 3}` and a `Set`
  wrapper. Requires lexer rule for `{...}` to distinguish dict
  literal (`{a: 1}`, `{}`) from set literal (`{1, 2, 3}`).
- **`Bytes` wrapper class** — bring the existing `bytes`
  module under the `b"..."` literal as a class.
- **Integer / Float split** — introduce a separate big-integer
  or fixed-precision class. Requires promotion rules.
- **`Class` / metaclass tower** — `Number.class`, methods on
  class objects themselves (`Number.methods()`), arbitrary
  class instantiation.
- **`respond_to?` / `method_missing` / `define_method`** —
  dynamic introspection and reflection.
- **Operator overloading on built-in classes** — currently
  forbidden by the monkey-patching rule. A future Epic could
  allow `import_op_override "Number" as MyNumber` style scoped
  override but is explicitly not in v0.59.
- **`tya lsp` rename-across-files** for the kind / module-style
  migration. v0.59 will hand-edit and rely on the autofix in
  `tya lint`.

## Implementation notes (informative)

The work to ship v0.59 is split into long-running phases driven
by the `/goal` skill, not a single one-release sprint. Rough
ordering:

1. **lexer**: verify `.` disambiguation (already in place), add
   a regression test bank.
2. **parser / AST**: no new node kinds — literals remain
   `IntLit` / `FloatLit` / `StringLit` / `ArrayLit` / `DictLit` /
   `BoolLit` / `NilLit`. The class-instance semantics is added
   in the checker / codegen, not the AST.
3. **checker**: reserve the six identifiers, reject
   subclass-of-primitive, reject monkey-patch-of-primitive,
   reject `kind(...)`, reject module-style removed APIs.
   Emit `TYA-E08XX` diagnostics.
4. **codegen**: emit operator desugaring with the static
   fast-path for known primitives; emit `tya_dispatch_method`
   for the dynamic path; emit `x.class` as a static lookup.
5. **runtime**: build the six singleton class objects at
   startup. Add a small dispatch table per wrapper class.
   Helpers: `tya_class_of(value) -> TyaClass *`, with all
   primitive cases lowered from the `kind` switch.
6. **stdlib**: delete `stdlib/string.tya`, `stdlib/array.tya`,
   `stdlib/dict.tya`. The wrapper-class method bodies are
   provided directly by the runtime (via the dispatch table)
   — the operations themselves remain the existing C helpers,
   only the surface (method instead of free function) changes.
7. **selfhost**: update `selfhost/v01/compiler.tya` to use the
   new method-style surface and re-prove the v01 stage-2 ==
   stage-3 fixed point.
8. **examples / tests**: hand-edit `examples/`,
   `tests/testdata/v01-v40/*.txtar` and similar to use
   `s.upper()` instead of `string.upper(s)`, `x.class.name`
   instead of `kind(x)`, etc.
9. **docs**: rewrite `docs/SPEC.md`, `docs/STDLIB.md`,
   `docs/API.md`, `docs/NAMING.md` for the new surface. Add
   `docs/v0.59/RELEASE_NOTES.md` summarising the migration.
10. **release flow**: version bump, Formula, VERSIONS, docs
    HTML, ROADMAP entry, brew tap sync.

Reverse-compatibility for an older `tya` toolchain (running a
v0.59-syntax `.tya` source through a v0.58 compiler) is **not**
provided. v0.59 is a breaking minor; users on v0.58 must update
both toolchain and sources together. The `tya lint --fix`
autocorrect is the recommended migration path.
