# Tya v0.1 Language Spec

この文書は Tya v0.1 の言語仕様である。

v0.1 の正規の実行経路は、Tya ソースを lex / parse / check し、C を出力し、
C コンパイラで実行可能ファイルにする経路である。

```text
Tya source -> lexer -> parser -> AST -> checker -> C emitter -> C compiler -> executable
```

Go interpreter、現行 `selfhost/*`、ASTMODE、legacy node string、self-host
bootstrap gate は v0.1 仕様の正ではない。削除しなくてもよいが、v0.1
実装完了まではメンテナンス対象外とする。

## Scope

v0.1 に含めるもの:

- `.tya` ファイル
- 2スペースインデントのブロック構文
- コメント
- 代入
- 複数代入
- 配列
- 辞書
- 添字アクセス
- 関数リテラル
- 関数呼び出し
- `if` / `elseif` / `else`
- `while`
- `for value in array`
- `for value, index in array`
- `for key, value of dictionary`
- `break` / `continue`
- `return`
- 複数戻り値
- `try`
- `error`
- `module`
- `import module_name`
- `module.member`
- 文字列補間
- v0.1 標準組み込み関数
- C へのコンパイルと実行

v0.1 に含めないもの:

- object
- class
- interface
- inheritance
- `self`
- `super`
- `@property`
- object method
- class method
- class field
- import alias
- dictionary member access
- set literal
- package manager
- async
- macro
- exception
- Go interpreter を正規実行経路にすること
- legacy node string 互換を仕様に含めること

## Source Files

Tya ソースファイルは `.tya` 拡張子を使う。

entry file は `import` と文を持てる。imported module file は、ファイル名と同じ
名前の `module` 宣言を1つ持つ。

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

```tya
# greeting.tya
module greeting
  hello = name -> "Hello, {name}"
```

## Lexical Structure

コメントは `#` から行末まで。

```tya
# comment
name = "Tya"
```

ブロックはインデントで表す。1インデントは2スペースである。lexer は
インデント増減に応じて `INDENT` / `DEDENT` token を出す。

token には identifier、integer literal、float literal、string literal、
newline、indentation token、および以下の operator / delimiter がある。

```text
= == != < <= > >= : , . + - * / % -> ( ) [ ] { }
```

以下の identifier は予約された意味を持つ。

```text
if elseif else while for in of break continue return
import module
true false nil and or not try
```

`print`、`panic`、`exit`、`error` などは keyword ではなく標準組み込み関数である。

## Names

名前は `docs/NAMING.md` に従う。

```text
変数・関数:             snake_case
private binding:        _snake_case
module / file:          snake_case
dictionary key:         snake_case
module member:          snake_case
constant:               SCREAMING_SNAKE_CASE
type / class name:      PascalCase  # v0.1 では予約
```

## Values

v0.1 の runtime value は以下。

```text
nil
boolean
integer
float
string
array
dictionary
function
error
module
```

v0.1 には object は存在しない。将来 class を導入するとき、object は class
instance として定義する。

## Strings

文字列は以下の escape をサポートする。

```text
\" \\ \n \t
```

文字列内の `{expression}` は補間される。

```tya
name = "Tya"
print "Hello, {name}"
```

## Dictionaries

辞書は key-value collection である。

```tya
user = { name: "komagata", age: 20 }
empty = {}
```

v0.1 の辞書 literal の key は `snake_case` identifier である。辞書の値は添字で
読む。

```tya
print user["name"] # ok
print user.name    # invalid
```

`.` は v0.1 では module member access 専用である。辞書に対する member access
は無効である。

## Statements

代入:

```tya
name = "Tya"
left, right = pair
items[0] = 10
user["name"] = "komagata"
```

式文:

```tya
print "hello"
```

条件分岐:

```tya
if score >= 90
  print "A"
elseif score >= 80
  print "B"
elseif score >= 70
  print "C"
else
  print "D"
```

`elseif` は `if` の後に0個以上書ける。`else` は最後に0個または1個だけ書ける。
`elseif` と `else` は対応する `if` と同じインデントに置く。`elseif` は
`else if` ではなく1語の reserved identifier である。

while loop:

```tya
count = 0
while count < 3
  print count
  count = count + 1
```

array iteration:

```tya
for value in items
  print value

for value, index in items
  print "{index}: {value}"
```

dictionary iteration:

```tya
for key, value of user
  print "{key}: {value}"
```

loop control:

```tya
while true
  break

while true
  continue
```

return:

```tya
return
return value
return value, err
```

## Expressions

primary expression:

```text
identifier
integer
float
string
true
false
nil
[items]
{ name: value }
(expression)
```

function literal:

```tya
name -> expression
left, right -> expression

name ->
  statement
```

call:

```tya
fn()
fn(arg)
fn(arg1, arg2)
module.member()
```

v0.1 では、通常の関数呼び出しは必ず `()` を使う。`fn arg`、
`fn arg1, arg2`、`fn arg1 arg2`、`print len keys user` のような
no-paren call chain は構文に含めない。

例外として、`print` だけは statement-level の表示構文として `print expression`
を許す。`print` は後続の1式だけを受け取る。

```tya
print "hello"
print len(items)
print len(keys(user))
print add(2, 3)
```

複数引数や入れ子呼び出しは `()` で明示する。

```tya
add(2, 3)
len(keys(user))
has(user, "name")
```

index access:

```tya
items[0]
dictionary["name"]
```

module member access:

```tya
module_name.member_name
```

unary expression:

```text
not expression
-expression
try expression
```

binary operator precedence は低い順に以下。

```text
or
and
== !=
< <= > >=
+ -
* / %
unary: not, -, try
call / member / index
primary
```

## Truthiness

`nil` と `false` は falsey である。それ以外の値は truthy である。

## Functions

関数は child scope を作る。

```tya
double = value -> value * 2

sum = values ->
  total = 0
  for value in values
    total = total + value
  total
```

block function は、明示的な `return` がない場合、最後の式を返す。関数は複数の
値を返せる。

```tya
parse_user = text ->
  if text == ""
    return nil, error("empty user")
  return { name: text }, nil
```

## Errors

`error("message")` は `message` を持つ error value を作る。v0.1 では `.`
が module member access 専用なので、message は `err["message"]` で読む。

```tya
err = error("file not found")
print err["message"]
```

`try expression` は関数内でのみ使える。対象式は `value, err` の2値を返す想定と
する。`err` が truthy なら現在の関数から `nil, err` を返す。`err` が falsey
なら `value` を式の値にする。

```tya
load_user = text ->
  user = try parse_user(text)
  user["name"]
```

## Modules

`import module_name` は、importing file と同じディレクトリの
`module_name.tya` を読む。

imported module file はファイル名と同じ名前の `module` 宣言を1つ持つ。

```tya
# greeting.tya
module greeting
  hello = name -> "Hello, {name}"
```

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

module member は `module_name.member_name` で読む。v0.1 の `.` は module
member access 専用である。

import alias は v0.1 に含めない。

## Standard Builtins

v0.1 標準組み込み関数は `docs/API.md` に定義する。v0.1 では必須でない便利関数を
標準組み込みに含めない。

## Execution

v0.1 の正規実行経路は、Tya ソースから実行可能ファイルを作って実行する経路である。
ユーザー向け CLI は Go のコマンドを要求しない。

## Command Line

v0.1 は以下のユーザー向けコマンドを持つ。

```sh
tya run file.tya [args...]
tya build file.tya -o output
tya version
```

`tya run` は一時的な実行可能ファイルを作り、それを実行し、実行後に一時ファイルを
削除する。これは Go の `go run` と同じ位置づけのコマンドであり、
インタプリター実行ではない。

`tya build` は実行可能ファイルを作り、指定された出力先に残す。`-o` が
指定されていない場合は、入力ファイルの basename から `.tya` を除いた名前で、
カレントディレクトリに実行可能ファイルを作る。

```sh
tya build hello.tya
# writes ./hello

tya build examples/hello.tya
# writes ./hello

tya build examples/hello.tya -o bin/hello
# writes ./bin/hello
```

`tya version` は Tya のバージョンを標準出力に表示する。

Go interpreter と Go 開発用コマンドは v0.1 のユーザー向け正規実行経路ではない。

## v0.1 Reference Implementation

v0.1 の参照実装としてメンテナンスする対象は以下。

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.1 specification tests
```

Tya 版 compiler は、v0.1 仕様が Go 版 compile-to-C 経路で完全に動作してから、
AST ベースで新規実装する。
