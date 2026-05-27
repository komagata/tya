---
layout: doc
title: 仕様
lang: ja
permalink: /ja/spec/
---

# Tya 言語仕様

状態: 現在のリポジトリ仕様。このページは `main` で保守されている言語表面を記述する。現在のパッケージ、ツール、並行処理、インターフェイス、標準ライブラリ統合の規則を含む。

## はじめに

Tya はインデントベースの、動的型付けの、C にコンパイルされる言語である。実装は意図的に小さく明示的に保たれている。ソースはトークン化され、AST にパースされ、検査され、C として出力され、Tya ランタイムとリンクされる。

Tya がユーザー向けに約束することは次の通りである。

- `tya format` による正規ソース整形
- 暗黙変換のない厳密な動的セマンティクス
- C へコンパイルするランタイムモデル
- 実行、ビルド、検査、整形、テスト、lint、ドキュメント生成、パッケージ、エディタ支援を担う、オールインワンの `tya` コマンド
- 安定したコードを持つ構造化診断
- 保守されたセルフホスト経路

この文書は言語、組み込み関数 surface、標準ライブラリ surface、パッケージ規則、ツール surface を規定する。

[`docs/STRICT_SEMANTICS.md`](../STRICT_SEMANTICS.md) の厳密セマンティクス規則表は、v1.0.0 の妥当性境界について規範的であり、各規則ファミリーの parser、checker、runtime、CLI、LSP、self-host の有効なカバレッジを記録する。

## 表記

例は通常の Tya ソースを使う。文法断片は説明用であり、完全なパーサ文法ではない。

```text
snake_case            変数、関数、メソッド、インポートパスセグメント
SCREAMING_SNAKE_CASE  定数
PascalCase            クラスとインターフェイス
```

「must」「must not」「may」「should」は、プログラムの妥当性または実装の振る舞いを記述する場合に規範的な意味を持つ。

## ソースコード表現

Tya ソースは UTF-8 テキストである。コンパイラは字句解析の前に CRLF 改行を LF に正規化する。ソースファイルは `.tya` を使う。

インデントがブロックを定義する。スペースがインデント単位である。タブはソースインデントと heredoc 本文インデントで禁止される。

```tya
if ready
  print("ready")
else
  print("not ready")
```

各物理行は、括弧付き呼び出し、配列リテラル、文字列リテラル、またはパーサとフォーマッタが受け入れる正規継続形式の内部にある場合を除き、1 つの論理行の一部である。

## 字句要素

### コメント

行コメントは `#` で始まり、行末まで続く。

```tya
# file header comment
name = "tya" # line-end comment
```

コメントは、整形、LSP hover、`tya doc` のために宣言や文へ付着できる。コメント配置規則は正規構文の一部である。

Tya は 3 つのソースコメント役割を認識する。

- ファイル先頭のファイルヘッダーコメント
- 後続の宣言または文へ直ちに付着する先行コメント
- 直前の文へ付着する行末コメント

明確な付着対象を持たない位置のコメントは不正である。本体がコメントだけのブロックは、実行可能または宣言的な本体項目を持たないため不正である。

### トークン

トークン語彙には、識別子、リテラル、インデントトークン、演算子、句読点が含まれる。

```text
= == != < <= > >= : , . ? @ + - * / % ->
( ) [ ] { }
& | ^ ~ << >>
```

空白はトークンを分離する。改行は文を終端し、インデントブロックを定義するため意味を持つ。

### 識別子

識別子は慣習上も現在の命名規則上も ASCII 志向である。公開変数、関数、メソッド、ファイル、インポートパス名は `snake_case` を使う。クラス名とインターフェイス名は `PascalCase` を使う。定数は `SCREAMING_SNAKE_CASE` を使う。

次の単語は、通常の名前がパースされる位置で予約されている。

```text
abstract and as await break case catch class continue default else elseif embed
extends false final for if implements import in interface module nil not or
override private raise return scope select self Self spawn static super true try
while with
```

一部の単語は文脈依存である。たとえば、`as` はインポートで意味を持ち、`extends` と `implements` はクラスおよびインターフェイスヘッダーで意味を持ち、`case`、`default`、`send`、`receive`、`timeout` は `select` 内で意味を持つ。

### リテラル

Tya には `nil`、真偽値、数値、文字列、バイト、配列、辞書のリテラルがある。

```tya
missing = nil
ready = true
count = 42
ratio = 3.14
name = "Tya"
data = b"abc"
items = [1, 2, 3]
user = { name: "komagata", age: 20 }
```

文字列リテラルは二重引用符を使う。文字列は `{...}` による補間をサポートする。

```tya
print("Hello, {user["name"]}")
```

三重引用符文字列と heredoc 形式は複数行テキストに使える。raw heredoc と byte heredoc は、文書化されたエスケープ動作を保持する。フォーマッタは、正規構文が書き換えを定義する場所を除き、複数行文字列を不可分なものとして扱う。

バイトリテラルは `b"..."` または byte heredoc 形式を使い、文字列ではなくバイト値を生成する。

整数リテラルは 10 進、16 進、2 進形式で書ける。浮動小数点リテラルは 10 進表記を使う。

## 値と種類

Tya は動的型付けである。値は実行時 kind を持つ。中核となる実行時 kind は次の通りである。

```text
nil
bool
number
string
bytes
array
dict
function
class
object
error
task
channel
resource
```

配列と辞書はミュータブルである。文字列とバイトは別の値 kind である。クラスは実行時値であり、オブジェクト値はクラスのインスタンスである。

プリミティブ値は、ランタイムラッパークラスと標準組み込みを通してメソッドを公開する。

```tya
print(" tya ".trim().upper())
print([1, 2, 3].len())
print({ name: "tya" }.keys())
print(value.class)
```

Tya は暗黙変換を行わない。数値、文字列、配列、辞書、関数、クラス、タスク、チャンネル、リソースを必要とする操作は、必要な kind の値を受け取らなければならず、そうでなければ実行時エラーを raise する。文書化された例外は、文字列補間や `to_s()` のような整形操作、および下記の明示された演算子の場合だけである。

## ブロック

ブロックは、ヘッダー行と増加したインデントレベルによって導入される、空でない、または空の文列である。

```tya
while count < 3
  print(count)
  count = count + 1
```

ブロックは、制御フロー文、関数本体、クラス本体、インターフェイス本体、`try` / `catch`、`scope`、`select` などの構文に現れる。

トップレベルソースは、インポート、宣言、代入、ファイル種別で許可された文からなる。クラスファイルはスクリプトファイルより制約が強い。

## ファイル種別

`.tya` ファイルの役割は、ファイル名と文脈によって決まる。

`snake_case` の `.tya` ファイルは、内容がクラス/インターフェイスファイル規則を満たさない限りスクリプトファイルである。スクリプトファイルは `tya run` のエントリファイルにでき、直接インポートすることもできる。インポートされた場合、そのトップレベル名はインポート束縛を通して公開される。

クラス/インターフェイスファイルは `snake_case` のファイル名を使い、ライブラリ専用であり、エントリファイルにはできない。クラス/インターフェイスファイルは、`base64.tya` が `class Base64` を宣言するように、ファイル名に対応する `PascalCase` 名の公開クラスまたは公開インターフェイスをちょうど 1 つ宣言しなければならない。`Base64.tya` のような PascalCase ファイル名はクラスファイルではない。

クラスファイルは、ディレクトリパッケージの一部として明示的に読み込まれる場合も、エントリスクリプトと同じディレクトリにある兄弟として暗黙に読み込まれる場合もある。スクリプトエントリは、同じディレクトリの `snake_case` クラス/インターフェイスファイルをインポートなしで見る。

このファイル名規約は、従来の PascalCase クラスファイル規約からのマイナーバージョンの言語/パッケージ規約変更である。既存の `Base64.tya` 形式のファイルは、対応する `snake_case.tya` 形式にリネームしなければならない。

## 正規構文 {#canonical-syntax}

Tya には正規構文がある。整形式のすべてのプログラムは 1 つのソース表現を持つ。そのため `tya format` は任意のスタイルツールではなく、言語表面の一部である。

正規構文は、インデント、空行、コメント付着、行折り返し、インポートグループ化、演算子まわりの空白、文字列リテラル形式、空コレクション形式、その他のソース形状の判断を扱う。フォーマッタは正規 serializer であり、style configuration を持たない。

中核規則:

- インデントは 2 spaces。タブは source 内で不正である
- column limit は 80。ただし単一の分割不能な atomic token は例外である
- コメントは file header comment、leading comment、line-end comment のいずれかで、明確な付着対象を持たなければならない
- 空行は AST shape で決まり、ユーザーの好みでは決まらない
- 複数行 call、array、dict、parameter list、operator chain、長い condition は formatter-defined continuation form を使う
- import が 1 つの場合は 1 行のままにし、連続する import が 2 つ以上ある場合は
  indented `import` block に整形する
- 1 つの式だけからなる関数・メソッド本体、または単一値の final
  `return` だけからなる本体は、rendered line が column limit に収まり、
  付着コメントが block body を必要としない場合 `-> expr` に整形される
- class body と interface body では隣接 member の間にちょうど 1 行の空行を置く。
  その空行は次の member に付いた leading comment の前に置き、最後の member
  の後には余分な空行を置かない
- class body は member category、static/instance、public/private、member name の順で並び、`initialize` は public instance method の先頭に置く
- `elseif` が正規綴りであり、`else if` は正規ではない
- `match` の `case _` は wildcard case であり、最後でなければならない
- empty collection form と empty `else` branch は formatter-defined shape に従う

実装は整形時に意味的な振る舞いを保持しなければならない。整形は冪等であり、プラットフォームをまたいで安定していなければならない。

## 宣言とスコープ

### 束縛

代入は束縛を作成または更新する。

```tya
name = "Tya"
count = count + 1
```

複数代入がサポートされる。

```tya
value, err = parse_user(text)
```

通常の束縛では、先頭の `_` は visibility の意味を持たない。トップレベルの privacy は名前の綴りでは表現しない。

定数は `SCREAMING_SNAKE_CASE` を使い、命名規則と代入規則によって定数として検査される。定数は再代入できず、定数束縛を通して heap-backed value を変更できない。

クラスメンバーの privacy には `private` キーワードを使う。private なクラスフィールド、クラス定数、メソッド、クラス変数、クラスメソッド、コンストラクタは `private` で宣言する。

```tya
class User
  private ROLE = "user"

  private id = 0

  private normalize = ->
    Self.ROLE + ":" + self.id.to_s()
```

### 埋め込みアセット

`embed` は、ビルド時にファイルから値を読み込むトップレベル束縛を宣言する。embed 宣言はソースファイルからの相対パスとして解決される。

```tya
embed "templates/index.html" as index_html
```

embed 変換はコンパイラ表面によって実装定義であり、通常の Tya 値を生成しなければならない。

### 関数

関数は値である。関数リテラルは `->` を使う。

```tya
greet = name -> "Hello, {name}"

double = value ->
  result = value * 2
  result
```

正規構文では、1 つの式だけの本体は one-line form を優先する。

```tya
answer = -> 42
```

`return value` だけを含む block も `-> value` に整形される。複数文の本体、付着コメントを持つ本体、one-line rendering が column limit を超える本体は block body のままにする。

呼び出しは常に括弧を使う。

```tya
print(greet("Tya"))
```

関数本体の最後の式は暗黙に返される。早期 return または複数戻り値には `return` を使う。

```tya
parse_user = text ->
  if text == ""
    return nil, error("empty user")
  return { name: text }, nil
```

パラメータはローカル束縛である。意図的に無視するパラメータには `_` を使える。

### クラス

クラスは実行時クラス値を宣言する。

```tya
class User
  initialize = name ->
    self.name = name

  label = ->
    "user:{self.name}"
```

インスタンスはクラスを呼び出すことで構築される。

```tya
user = User("komagata")
print(user.label())
```

`initialize` はコンストラクタフックである。インスタンスメソッドは `self` を受け取る。インスタンスフィールドは `self.<name>` への代入で作成される。

Tya は次をサポートする。

- `extends` による単一クラス継承
- `super(...)` によるコンストラクタおよびメソッド委譲
- `private` メンバー
- `SCREAMING_SNAKE_CASE = value` で宣言するクラス定数
- `static` クラスメソッドとクラス変数
- `abstract class` と抽象メソッド
- `final class`
- 明示的なメソッド override 検査のための `override`
- `.class` による実行時クラス検査
- ランタイムで文書化された `class`、`class_name`、`name`、`parent` などの読み取り専用クラスメタデータメンバー

クラス定数はクラスが持つ immutable なメンバーである。定義クラス内では `Self.NAME` でアクセスするのが canonical である。public なクラス定数は `pkg.Class.NAME` として外部から読める。private なクラス定数は定義クラスの外から読めない。`static NAME: ...` はクラス変数の書き方であり、canonical な定数形式ではない。

```tya
class Admin extends User
  initialize = name ->
    super(name)

  override label = ->
    "admin:{self.name}"
```

クラスファイルは `snake_case` の `.tya` ファイルである。ファイル名に対応する `PascalCase` 名の公開クラスまたは公開インターフェイスをちょうど 1 つ宣言しなければならない。private な補助クラスとインターフェイスも宣言できる。クラスファイルはライブラリファイルであり、エントリスクリプトとして実行できない。

クラスファイル内の追加クラスはそのファイルに private である。同じディレクトリパッケージ内であっても、他のファイルからは見えない。

### インターフェイス

インターフェイスは明示的な契約であり、積み重ね可能な振る舞い単位である。

```tya
interface Named
  name = ->

  label = ->
    self.name()
```

インターフェイスには次を含められる。

- 本体を持たないインスタンスメソッド要求
- デフォルトインスタンスメソッド
- フィールド宣言
- 引数なしの `initialize` フック

インターフェイスには static メンバー、private メンバー、ネストしたクラス、ネストしたインターフェイスを含められない。`Self` はインターフェイスメソッド内で不正である。

クラスは `implements` で実装するインターフェイスを列挙する。

```tya
interface Timestamped
  created_at = nil

  initialize = ->
    self.created_at = Time().now()

class Account implements Named, Timestamped
  initialize = name ->
    self.name_value = name
    super()

  name = ->
    self.name_value
```

クラスが同名メソッドを定義しない場合、デフォルトメソッドは継承される。クラスメソッドはインターフェイスデフォルトに優先する。インターフェイスデフォルトは宣言された `implements` 順に積み重なり、`super()` を呼べる。

インターフェイスフィールドはインスタンス状態に寄与する。複数のインターフェイスを実装するクラスは、衝突するフィールド定義を受け取ってはならない。クラスコンストラクタが initializer フックを持つインターフェイスを実装する場合、インターフェイス初期化チェーンを実行したい位置で正確に `super()` を呼ばなければならない。

インターフェイス衝突規則は厳密である。

- 重複する要求は 1 つの要求へ畳み込まれる
- デフォルトメソッドは要求を満たせる
- 同じメソッドに対する無関係なデフォルトは、クラスがそのメソッドを override しない限り曖昧である
- arity の衝突はエラーである
- initializer の順序は決定的であり、クラス継承が新たに実装されたインターフェイスより先に来る

クラスファイル内で宣言されたインターフェイスは、名前が `_` で始まらない限り、パッケージの公開名としてエクスポートされる。

`Comparable` は標準の順序付けプロトコルである。クラスは `compare(other)` を提供することでこれを実装する。`compare(other)` は、レシーバーが `other` より前、等しい、または後にソートされる場合に、それぞれ負数、ゼロ、正数を返す。数値と文字列はプリミティブ値として `Comparable` に適合する。順序演算子 `<`、`<=`、`>`、`>=` は既存のプリミティブ動作を保ち、ユーザー定義の `compare` へディスパッチしない。

`Equatable` は標準のドメイン等価性プロトコルである。クラスは `equal?(other)` を提供することでこれを実装する。`equal?(other)` は真偽値を返さなければならない。プリミティブ値は `equal?` を公開する。スカラープリミティブは `==` に従い、配列と辞書は deep equality を使う。`==` 演算子とトップレベルの `equal(left, right)` は既存の動作を保ち、ユーザー定義の `equal?` へディスパッチしない。

`Stringable` は標準の人間可読フォーマットプロトコルである。クラスは `to_s()` を提供することでこれを実装する。`to_s()` は文字列を返し、通常のフォーマット用途では副作用を持たないべきである。Number、String、Array、Dict、Boolean、Nil は、タグ付きランタイム表現や `value.class` の振る舞いを変えずに、プリミティブ値として `Stringable` に適合する。`Stringable` は構造化シリアライズプロトコルではない。データツリーには `Serializable.to_data()` を使う。

標準ライブラリは、iteration、sequence、I/O、構造化データのための protocol interface も定義する。

- `Iterator` は `has_next()` と `next()` を要求する
- `Iterable` は `iter()` を要求し、`sequence()` を提供する
- `Sequence implements Iterable` は lazy-style の `map(fn)`、`filter(fn)`、`take(n)`、`drop(n)`、`reduce(initial, fn)`、`to_a()` を提供する
- `Readable` は `read(size)` を要求する
- `Writable` は `write(data)` を要求する
- `Closable` は `close()` を要求する
- `Flushable` は `flush()` を要求する
- `Serializable` は `to_data()` を要求する

配列、辞書、文字列はプリミティブ値として `Iterable` に適合する。`for ... in` はプリミティブ iterable を直接消費し、ユーザー定義 iterable は `iter()` を通して消費する。I/O protocol interface は `io` や `net/socket` などの関連する標準ライブラリパッケージで定義される。それらは共有 stream behavior を文書化し、メソッドが一致する concrete reader、writer、socket、server class によって実装される。

## 式

式は値を計算する。

### 一次式

一次式には、識別子、リテラル、括弧付き式、関数リテラル、インデックス、メンバーアクセス、呼び出し、`self`、`Self`、`super` が含まれる。

```tya
user["name"]
items[0]
user.label()
User("komagata")
self.name
super(name)
```

`self` はインスタンスメソッドとコンストラクタの内側で利用できる。`Self` は、それが有効なクラス文脈で現在のクラスを参照する。`super(...)` は文脈に応じて、親コンストラクタ、親メソッド、または次に積み重ねられたインターフェイスメソッドへ委譲する。

関数リテラルは lexical closure である。関数リテラルは外側の関数本体からパラメータとローカル束縛を読める。キャプチャは関数リテラルが評価された時点で作成される値スナップショットである。配列、辞書、オブジェクト、関数、リソース、タスクのようなヒープ上の値は、deep copy ではなく値としてキャプチャされる。トップレベル名はキャプチャされず、モジュールまたはグローバル lookup を使い続ける。

関数本体は外側の束縛へ書き戻せない。外側束縛の直接再代入は不正であり、キャプチャされた外側束縛を通したインデックス代入またはメンバー代入も不正である。関数がその値を変更する意図を持つ場合、ミュータブルな状態は明示的なパラメータとして渡す。

関数リテラルが評価されるたびに、独立した closure environment が作られる。

```tya
make_adder = base ->
  value -> base + value

add_two = make_adder(2)
add_ten = make_adder(10)

print(add_two(3))
print(add_ten(3))
```

closure 作成後に元のローカルを再代入しても、キャプチャ済みの値は変わらない。

```tya
make_label = name ->
  label = -> name
  name = "changed"
  label

print(make_label("first")())
```

キャプチャされた束縛を通した mutation は不正である。mutation を意図する場合、closure はそのミュータブル値をパラメータとして受け取らなければならない。

```tya
make_bad = items ->
  ->
    items[0] = "changed" # invalid: cannot mutate captured binding
```

### 演算子

Tya は算術、比較、論理、ビット演算子をサポートする。

```text
or
and
not
== != < <= > >=
| ^ &
<< >>
+ -
* / %
```

論理演算子は単語 `and`、`or`、`not` を使う。

```tya
if ready and not disabled
  print("ok")
```

算術演算は、文書化されたプリミティブメソッドまたは演算子ケースが別のことを述べる場合を除き、数値を要求する。`+` はどちらかのオペランドが文字列の場合に文字列変換を通して整形し、2 つのバイト値を連結する。文字列補間は `Stringable` surface で埋め込まれた値を整形する。`nil` 算術は不正である。

ビット演算子は整数互換の数値を要求する。

等価演算子は暗黙変換なしに任意の 2 つの実行時値を比較できる。順序演算子 `<`、`<=`、`>`、`>=` は数値を要求する。

### コレクション

配列は角括弧リテラルと整数インデックスを使う。

```tya
items = ["a", "b"]
items.push("c")
print(items[0])
```

辞書は波括弧リテラルを使う。辞書リテラル内の識別子キーは文字列キーとして保存される。

```tya
user = { name: "komagata", age: 20 }
print(user["name"])
user["admin"] = true
```

辞書ブロック形式と空コレクション形式はフォーマッタによって正規化される。

配列、文字列、バイトのインデックスは整数でなければならない。辞書と error 値のインデックスは文字列でなければならない。存在しない辞書キーと範囲外の配列または文字列インデックスは `nil` を返す。コレクションでない対象へのインデックスは不正である。

### エラー式

`try` は関数本体内で式として使える。`catch` 分岐は raise された値を受け取る。

```tya
load_name = path ->
  try
    read_file(path).trim()
  catch err
    "guest"
```

### 並行処理式

`spawn` はタスクを開始し、タスク値を返す。`await` はタスクを待ち、その結果を返すか再 raise する。

```tya
task = spawn work(21)
print(await task)
```

チャンネルと sync リソースは、標準ライブラリ節で規定されるメソッドを持つ、標準ライブラリに支えられたランタイム値である。

## 並列性と並行性

Tya はタスク、scope、チャンネル、sync リソース、`select` を通して構造化並行性を公開する。

タスクは `spawn` によって作られる軽量なランタイム値である。`await` はタスクを join する。完了済みタスクを await すると、キャッシュされた結果を返すか、キャッシュされたエラーを再 raise する。

`scope` は、その内部で spawn されたタスクに対する構造化された lifetime を定義する。scope は領域を離れる前に子タスクを待つ。

チャンネルと sync リソースはランタイムによって実装され、標準ライブラリのクラスとメソッドを通して公開される。`select` はチャンネル送信、受信、timeout、default 分岐をまたいで待つ。

ランタイムは、対象プラットフォームとランタイムがサポートする場合にタスクを並列実行してよい。プログラムの正しさは、言語または標準ライブラリが順序保証を文書化している場合を除き、特定のスケジューリング順序に依存してはならない。

## 文

### 式文

呼び出しやその他の有用な式は文として現れてよい。

```tya
print("hello")
save_user(user)
```

### 代入文

代入は束縛、フィールド、またはインデックスされたコレクション項目を更新する。

```tya
name = "Tya"
self.name = name
items[0] = "first"
user["admin"] = true
```

複数代入は右辺を評価し、対応する左辺名へ束縛する。

### If 文

`if`、`elseif`、`else` はブロックを選択する。

```tya
if age >= 20
  print("adult")
elseif age >= 13
  print("teen")
else
  print("young")
```

`elseif` が正規の綴りである。`else if` は正規 Tya ではない。
`nil` と `false` だけが falsey である。`0`、`""`、空配列、空辞書を含むその他の値はすべて truthy である。

### While 文

`while` は条件が truthy である間繰り返す。

```tya
while count < 3
  print(count)
  count = count + 1
```

### For 文

`for ... in` は iterable 値を消費する正規の方法である。配列は要素を yield し、文字列は文字を yield し、辞書は `{ key: key, value: value }` の entry 辞書を yield し、ユーザー値は `iter()` を公開することで参加する。2 つ目の束縛がある場合、ゼロ始まりの index を受け取る。

```tya
for item in items
  print(item)

for item, index in items
  print("{index}: {item}")

for entry in user
  key = entry["key"]
  value = entry["value"]
  print("{key}: {value}")
```

`break` は最も近いループを抜ける。`continue` は次の iteration へ進む。

### Match 文

`match` は式を case pattern と比較し、1 つの `case` ブロックを選択する。`case _` はワイルドカード case であり、最後の case としてのみ正規である。

```tya
match value
case "ok"
  print("ok")
case _
  print("other")
```

### Return 文

`return` は現在の関数またはメソッドを終了する。0 個、1 個、または複数の値を返せる。

```tya
return
return value
return value, err
```

### Raise、Try、Catch 文

`raise` は値を raise する。`try` はブロックを実行し、raise された値を `catch` で処理する。

```tya
try
  save_user(user)
catch err
  print("save failed: {err}")
```

### Scope 文

`scope` は構造化並行性の領域を定義する。scope 内で spawn されたタスクは、scope を抜ける前にランタイム scope 規則に従って join される。

```tya
scope
  task = spawn work()
  print(await task)
```

### Select 文

`select` はチャンネル操作、timeout、default 分岐を待つ。

```tya
select
case value = receive ch
  print(value)
case send ch, next
  print("sent")
timeout 1
  print("timeout")
default
  print("none")
```

正確なチャンネルメソッドと sync primitive は標準ライブラリ節で定義される。

## 組み込み関数

Tya の public builtin surface は意図的に小さい。file、directory、path、process、stream、bytes、random、compression、digest、socket、compiler、collection helper 操作は class-style standard-library API として公開される。これらの class を実装するために low-level runtime intrinsic が内部に存在してよいが、public standalone builtin ではない。

Public builtins:

```text
print(value)
println(value)
error(message)
exit(status)
args()
env(name)
```

low-level intrinsic name ではなく、`File().read(path)`, `File().append(path, text)`,
`Dir().list(path)`, `Path().expand_user(path)`, `Process().cwd()`,
`Process().chdir(path)`, `Io().open(path, mode)`, `Reader#read(size)`,
`Writer#write(value)`, `Random().int(min, max)`, `compress.Gzip().compress(value)`,
`Digest.sha256(value)`, `Socket.connect(host, port, options)`,
`Lexer().lex(source)`, `Parser().parse(source)`, `Checker().check(source)`,
`Format().format(source)` などの standard-library API を使う。conversion と
collection helper は `value.to_s()`, `value.to_i()`, `dict.delete(key)`,
`dict.keys()`, `items.pop()` のような receiver method を使う。

標準ライブラリ API は、ユーザーコードと同じ `import` 構文でインポートされる。

## 用語

現在の Tya documentation では、次の用語を規範的に使う。

```text
language feature             Tya に組み込まれた構文または semantics
built-in function            import なしで利用できる関数
built-in class               import なしで利用できる class。現在は存在しない
user package                 snake_case class/interface file の import 可能な directory
user library                 user package の再利用可能な directory tree
standard-library package     Tya に同梱され、通常通り import される .tya source
bundled library              toolchain と一緒に配布される library または support file
native-backed stdlib module  runtime または host code に支えられた import 可能 stdlib API
package                      tya.toml で宣言される versioned dependency unit
package tool                 tya tool で実行される [tools] entry
```

language feature は import されず、shadow できない。standard-library package は標準ライブラリ節で規定される。それらは import される package であり、builtin ではない。

## インポートとパッケージ

### インポート構文

インポートはトップレベルで、他の宣言や文より前に現れる。

```tya
import greeting
import net/http/client
import net/http/* as http
```

複数の import がある場合、正規構文では 1 つの `import` header の下に group 化する。

```tya
import
  greeting
  net/http/client
  net/http/* as http
```

インポートパスはスラッシュ区切りの `snake_case` セグメントである。相対ファイルシステムパス、絶対パス、空セグメント、`snake_case` ではないセグメントは不正である。package-wide import には final `/*` suffix を明示する。`*` は `path/*` の最後の segment としてだけ有効であり、`*`、`base64*`、`base64/*/foo`、`base64/**` は不正である。

### ディレクトリパッケージ

ディレクトリパッケージは、インポートパスによって解決される、`snake_case` クラス/インターフェイスファイルを含むディレクトリである。少なくとも 1 つのクラス/インターフェイスファイルを含まなければならず、パッケージの leaf にスクリプトファイルを含んではならない。

alias なしの wildcard ディレクトリインポートは、公開クラス名とインターフェイス名を直接公開する。

```tya
import net/http/*

server = Server()
```

alias 付き wildcard ディレクトリインポートは namespace 束縛を公開し、公開名を裸の名前としてインポートしない。

```tya
import net/http/* as http

server = http.Server()
```

同じディレクトリパッケージ内では、兄弟の公開クラスがインポートなしの裸の `PascalCase` 名で見える。

directory package の public API は、その `snake_case` class/interface file に含まれる public class と interface の集合である。class または interface は PascalCase name が filename に対応する場合に public である。class file 内の追加 class はその file に private である。

### ユーザーライブラリ

user library は再利用を意図した importable source の directory tree である。manifest は不要である。library root は `TYA_PATH` で利用可能にできる。

```sh
TYA_PATH=libs/web tya run app.tya
```

`TYA_PATH` entry は import root であり、relative import syntax ではない。user library 内の source は、application が使うものと同じ import path を使うべきである。

### 解決順序

インポートは次の順序で解決される。

1. インポート元ファイルのディレクトリ
2. `tya.lock` の manifest 宣言依存
3. `TYA_PATH` に列挙されたディレクトリ。左から右
4. バンドルされた `lib/` ディレクトリ

最初に一致したファイルまたはパッケージディレクトリが採用される。ローカルアプリケーションインポートは、パッケージ依存、`TYA_PATH`、標準ライブラリインポートを shadow してよい。パッケージ依存は `TYA_PATH` と標準ライブラリインポートを shadow してよい。

### パッケージマニフェスト

`tya.toml` はパッケージメタデータ、依存、native wrapper、パッケージ提供ツールを宣言する。`tya install` は依存を解決して `tya.lock` を書く。Git 依存とローカルパス依存がサポートされる。現在、中央パッケージレジストリと `tya publish` コマンドはない。

package は再利用可能な Tya code の versioned distribution unit である。package code は通常、`src/` の下で importable source を公開する。application は manifest dependencies を通して package を利用する。

```toml
[dependencies]
my_lib = { git = "https://github.com/example/my_lib", tag = "v0.1.0" }
local_lib = { path = "../local_lib" }
```

`tya.lock` は解決済み dependency source を記録し、application では commit すべきである。

native package metadata は `[native]` の下に置かれる。native path は package root からの相対パスである。`tya build`、`tya run`、`tya test` は、宣言された C source を generated C、Tya runtime、include directory、`pkg-config` flags、`cflags`、`ldflags` と一緒に compile する。native wrapper function は Tya runtime ABI を使い、その package 内では predeclared function のように呼ばれる。

package-provided tool は `[tools]` の下に置かれ、`tya tool` で実行される。package tool は global install でも shell task でもない。locked dependency または explicit one-shot git/path source から実行される。

## プログラム実行

スクリプトファイルは小文字の `.tya` ファイルであり、`tya run`、`tya build`、`tya emit-c` のエントリファイルとして使える。

```sh
tya run hello.tya
tya build hello.tya -o hello
tya emit-c hello.tya
```

クラスファイルはライブラリ専用であり、エントリファイルにはできない。

Tya は native 実行のために C へコンパイルするパイプラインを使う。`tya run` は一時 native 実行ファイルをコンパイルし、実行し、その一時実行ファイルを削除する。`tya build` は再利用可能な実行ファイルを書く。`tya emit-c` は Tya ソースから生成された C プログラムを表示または書き出す。生成された C は Tya ランタイムへリンクされる。

デフォルトの native target は Tya-managed Zig toolchain を `zig cc` として使う。`[native]` の native パッケージメタデータは、C ソース、ヘッダー、include ディレクトリ、`pkg-config` フラグ、コンパイラフラグ、リンカフラグをビルドへ寄与する。

WASM build target は、サポートされる場所で利用できる。native パッケージは未サポートの WASM target で拒否される。`tya run` は native 専用のままである。

## クロスコンパイル {#cross-compilation}

クロスコンパイルは `tya build` の `--target` で選択する。native target がデフォルトであり、Tya-managed Zig toolchain を `zig cc` として使う。WebAssembly target はプログラムを実行せず、別の実行環境向けの artifact を生成する。

現在の target は次を含む。

- `native`: ホスト native executable target
- `wasm32-wasi`: WASI runtime 向けの WASI `.wasm` artifact
- `wasm32-browser`: ブラウザ向け `.wasm` artifact と JavaScript loader

代表的なコマンド:

```sh
tya build --target native src/main.tya -o app
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
```

`tya doctor wasm` は WebAssembly build 環境と選択された Zig path/version を報告する。`tya doctor native` は native build 環境と選択された managed `zig cc` path/version を報告する。native パッケージメタデータは native build に C source と linker flag を寄与できるが、未サポートの native requirement を持つ package は未サポートの WebAssembly target で拒否される。

WebAssembly build は compile-to-C backend を保持し、native build と同じ Zig resolver を使う。最初の WebAssembly target layer は stdout-oriented smoke programs をサポートする。browser build は filesystem と process-oriented imports も拒否する。`tya run` は native 専用であり、WebAssembly artifacts を実行しない。

## 組み込みツール {#builtin-tools}

`tya` コマンドはオールインワンの toolchain である。同じ binary が、コンパイラ、フォーマッタ、language server、test runner、linter、documentation generator、package manager、project scaffolder、task runner、doctor commands、package tool runner を含む。

中核コマンドは次を含む。

```text
tya run
tya build
tya emit-c
tya check
tya format
tya test
tya cover
tya lint
tya lsp
tya doc
tya new
tya task
tya install
tya update
tya add
tya remove
tya outdated
tya tool
tya doctor
tya embed
tya version
```

ツールコマンドは、適用可能な箇所で同じパーサ、checker、formatter、package resolver、diagnostic 規約を共有する。これにより、各コマンドを別々の frontend として扱うのではなく、コマンドの振る舞いが言語仕様と揃う。

`tya run` はスクリプトファイルを一時 native 実行ファイルとしてビルドして実行する。ファイル名の後ろの引数は `args()` を通してプログラムへ渡される。

```sh
tya run src/main.tya
tya run examples/args.tya first second
```

`tya build` は再利用可能な実行ファイルまたは target artifact をビルドする。サポートされる native target と WebAssembly target 用に `--target` を受け付け、出力先には `-o` を使う。

```sh
tya build src/main.tya -o bin/app
tya build --target native src/main.tya -o app
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
```

`tya emit-c` は、検査、debugging、外部ビルドパイプライン用に生成された C プログラムを出力する。

```sh
tya emit-c src/main.tya
tya emit-c src/main.tya > build/main.c
```

`tya check` はプログラムを実行したり C コンパイラを呼び出したりせず、ソースを parse し、import を load し、検証する。editor と CI のための高速な semantic validation command である。

```sh
tya check src/main.tya
tya check tests/user_test.tya
```

`tya format` は正規ソースを表示する。`tya format -w` はファイルを in place に書き換える。整形は冪等であり、Tya source の canonical serializer である。

```sh
tya format src/main.tya
tya format -w src/main.tya
tya format -w src tests
```

`tya test` は標準 `unittest` surface を使ってテストを発見し実行する。現在の project、directory、特定の test file を実行できる。`--cover` で coverage collection を有効にする。

```sh
tya test
tya test tests
tya test tests/user_test.tya
tya test --cover
tya test --cover --profile .tya/coverage/profile
```

`tya cover` は coverage profile を報告する。デフォルトの report は人間可読であり、machine 用の JSON と閲覧用の HTML も利用できる。

```sh
tya cover
tya cover --format=json
tya cover html -o .tya/coverage/index.html
tya cover --profile .tya/coverage/profile --min 80
```

`tya lint` は安定した linter diagnostics を報告する。lint diagnostics は warning であり、program の validity error ではない。autofix と text、JSON、SARIF 出力をサポートする。JSON finding は `code`、`title`、`message`、`path`、`line`、`col`、`autofixable`、`doc_url` を含む。SARIF と LSP diagnostics は同じ stable rule code、title、help URL、warning severity を使う。public rule documentation は `docs/lint.md` と `https://tya-lang.org/lint.html#tyal000N` に置く。

現在の lint rules は次のとおり。

```text
TYAL0001 unused local binding              autofix
TYAL0002 dead code after return or raise
TYAL0003 redundant constant if             autofix
TYAL0004 deeply nested block
TYAL0005 long function body
TYAL0006 suspicious for-index binding order
TYAL0007 unused function parameter
TYAL0008 shadowed binding
```

```sh
tya lint src
tya lint --fix src
tya lint --format=json src
tya lint --format=sarif src > lint.sarif
```

`tya lsp` は標準入力と標準出力上の JSON-RPC で language server を実行する。editor は、ユーザーが直接操作する terminal command ではなく、長時間動作する subprocess として起動する。

```sh
tya lsp
```

`tya doc` は public top-level function、class、module、interface に付いた leading source comment からドキュメントを抽出する。path なしでは `src/` を scan する。text view、JSON report、静的 HTML site を生成できる。

```sh
tya doc
tya doc src
tya doc --json src
tya doc --html ./out src
```

JSON report は `version`、`items`、`diagnostics` を含む。各 item は name、kind、signature、raw Markdown comment、rendered text、source path と line、import された package surface 経由で含まれた場合の `reexported_from` を持つ。`tya doc` は import をたどって public re-export を含め、import cycle は hang せず diagnostic として報告する。

documentation diagnostic は stable な `TYADOC` code を使う。orphan doc comment、duplicate public documentation name、未サポート Markdown body、import cycle は text/HTML では stderr に出力され、JSON では payload に含まれる。error diagnostic は output 生成後に exit status 1、argument、path、I/O、lex、parse の致命的失敗は exit status 2 になる。

`tya new` は native package scaffold を含む新規 project と library を scaffold する。

```sh
tya new app
tya new --template lib mylib
tya new --template lib --native my_native_lib
```

`tya task` は `tya.toml` の `[tasks]` で宣言された project-local shell task を一覧表示および実行する。manifest は現在ディレクトリから親へ探索され、command は project root を working directory として実行される。string task は 1 つの `/bin/sh -c` command として実行される。array task は各 entry を順に実行し、最初の失敗で停止する。table task は `cmds = [...]` を使い、`parallel = true` の場合は各 command を並行実行する。

table task は `depends_on = ["build", "lint"]` で依存 task を宣言できる。依存は選択された task の前に、書かれた順序で、1 invocation につき 1 回だけ実行される。依存 cycle と未知の依存は、どの task command も実行する前に診断される。

table task の `env = { KEY = "value" }` は、その task だけの environment override である。依存 task は自分自身の `env` を使い、下流 task の `env` は漏れない。

`tya task <name> --watch` は task を 1 回実行し、監視対象の project file が変わるたびに再実行する。default では `.tya` file、`tya.toml`、存在する `src/`、`tests/`、`lib/`、`examples/` 配下を監視し、`.git/`、`node_modules/`、`_site/`、一般的な build output directory、hidden cache directory を無視する。table task の `watch = [...]` は default 監視対象を上書きし、`ignore = [...]` は ignore glob を追加する。`--watch` は `--` より前では task runner flag として消費され、`--` より後では task command へ渡される。

```sh
tya task
tya task run
tya task lint --fix
tya task dev --watch
```

`tya install` は `tya.toml` で宣言された依存を解決し、取得または materialize して、`tya.lock` を書く。

```sh
tya install
```

`tya update` は manifest に従って lock された依存バージョンを更新する。

```sh
tya update
tya update tya-sqlite
```

`tya add` は git、tag、revision、branch、path、dev dependencies を含む manifest dependencies を追加する。

```sh
tya add https://github.com/komagata/tya-sqlite --tag v0.1.0
tya add https://github.com/komagata/tya-raylib --branch main
tya add ../local-package --path
tya add --dev https://github.com/example/dev-tool --rev abc123
```

`tya remove` は manifest dependencies を削除する。

```sh
tya remove tya-sqlite
tya remove --dev dev-tool
```

`tya outdated` はより新しい利用可能バージョンがある依存を報告する。

```sh
tya outdated
```

`tya tool` はパッケージ提供ツールを一覧表示および実行する。ローカルパスまたは pin された git source からの one-shot tool execution もサポートする。

```sh
tya tool --list
tya tool format_docs --check
tya tool package_name:format_docs --check
tya tool --git https://github.com/example/tya-tools --tag v1.2.0 format_docs
tya tool --path ../tya-tools format_docs
```

`tya doctor` は native と WebAssembly build のための host environment details を報告する。

```sh
tya doctor
tya doctor native
tya doctor wasm
```

`tya embed` は embedded asset handling を検査または支援する。

```sh
tya embed --list src/main.tya
tya embed --list --format=json src/main.tya
```

`tya version` はインストールされた Tya バージョンを表示する。

```sh
tya version
```

## 検証コマンド

検証コマンドはソースを検査し、特定の contract を満たすかを報告する。検証コマンド自体は言語構文や標準ライブラリ挙動を定義しない。

検証コマンドは `tya format --check`、`tya check`、`tya lint`、`tya test`、将来の `tya verify` を含む。`tya run` と `tya build` は diagnostics と exit-code 規約を共有してよいが、実行および build command であり、検証コマンドではない。

`tya format --check` は source files が canonical Tya formatting に一致しているかを検査する。これは `tya format` がそのファイルを変更するかに答える。ファイルを書き換えてはならない。

`tya check` は C emission や execution の前に、source files が有効な Tya program であるかを検査する。lexical analysis、parsing、semantic checking、requested program に必要な import loading を含む。C emission、C compiler invocation、executable creation、program execution、unit test execution、lint rules は含まない。

`tya lint` は language validity に必須ではない rule を検査する。lint rule だけに失敗する program は有効な Tya program である。lint rules は built-in、project configuration、または将来の tooling で追加される rule であってよい。

`tya test` は unittest-based tests の execution entry point である。passed tests、failed assertions、サポートされる場合の skipped tests、runtime errors、test discovery errors を報告する。

`tya verify` は標準 verification pipeline のために予約される。その順序は次である。

```text
format --check -> check -> lint -> test
```

初期実装は、その時点で存在するコマンドだけを実行してよい。`tya verify` が存在するまでは、CI は次を直接実行できる。

```sh
tya format --check .
tya check .
```

検証コマンドは安定した exit-code meaning を使う。

```text
0  verification passed
1  verification failed
2  command usage error
3  internal tool error
```

検証コマンドは明示的な file と directory target を受け付ける。directory target は、その command に意味のある `.tya` source files を再帰的に選択する。target がない場合、より強い既存 convention を持つ command を除き、現在の directory を default target とする。

人間向け検証出力はデフォルトで簡潔である。failure は command name、file path、利用可能な場合の line と column、利用可能な場合の短い rule または diagnostic name、actionable message を含むべきである。multi-file command は実用的な場合、通常の verification failure の後も続行し、最後に summary を報告するべきである。

`--quiet`、`--verbose`、`--json` は一貫した verification behavior のために予約される。`--json` は human-readable output と同じ pass/fail meaning と exit codes を保持する。

検証コマンドは checking と rewriting を区別する。`tya format` はファイルを書き換えてよい。`tya format --check`、`tya check`、`tya lint`、`tya test`、`tya verify` はデフォルトではファイルを書き換えない。automatic lint fixes には `--fix` のような明示的 option が必要である。

## 単一バイナリ配布

Tya は 1 つの主要な `tya` binary として配布される。この binary は toolchain entry point を含み、リリースに同梱される標準ライブラリと C runtime files を使う。

1 binary モデルは、言語の運用設計の一部である。通常の Tya 作業のために、ユーザーが別々の formatter、test runner、LSP server、doc generator、package manager、build driver 実行ファイルを必要とするべきではない。

リリースは標準ライブラリ、C runtime sources、editor assets、examples、installation metadata などの support files を含んでよいが、コマンド表面は単一の `tya` executable を中心とする。

## エラーと診断

Tya には関連する 2 つのエラーシステムがある。

- プログラムエラーのための言語レベルの `raise`、`try`、`catch`
- 不正なソースと tool failure のための compiler および tool diagnostics

compiler diagnostics は `TYA-E0015` のような安定したコードを使い、linter diagnostics は `TYAL0001` のような安定したコードを使う。diagnostics は actionable なメッセージを含むべきであり、実用的な場合は hint と documentation URL も含むべきである。

runtime kind errors、不正な操作、失敗した assertions、失敗した I/O、native wrapper errors は、利用される API に応じて Tya error values または raised runtime errors として表現される。

## 標準ライブラリ

標準ライブラリは `lib/` の下で Tya に同梱され、ユーザーファイルやパッケージと同じ import 構文でインポートされる。

標準ライブラリは言語配布物の一部である。公開 surface は、`lib/` 以下の `snake_case` class/interface file から import 可能な PascalCase package class と interface の集合である。標準ライブラリ import は、local package、lock された package dependency、`TYA_PATH` entries の後に解決される。

public な標準ライブラリ class、interface、user-facing method は source doc comment を持つ。生成される stdlib API documentation はそれらの comment から `tya doc` で作られる。例えば `tya doc --json lib` は package path、signature、rendered comment、source path、source line を含む machine-readable reference を出力する。

現在の標準ライブラリ surface:

```text
base64/Base64              Base64 encode/decode helpers
binary/Reader              binary input reader
binary/Writer              binary output writer
channel/Channel            native channel value
cli/Cli                    command-line option parser and usage formatter
collections/Deque          double-ended queue
collections/PriorityQueue  priority queue
collections/Queue          FIFO queue
collections/Set            set collection
collections/Stack          LIFO stack
color/Color                RGBA color value and conversions
compiler/ast/Ast           compiler AST helpers
compiler/checker/Checker   compiler checker helpers
compiler/format/Format     compiler formatter helpers
compiler/lexer/Lexer       compiler lexer helpers
compiler/parser/Parser     compiler parser helpers
compress/Codec             compression codec interface
compress/Gzip              gzip compression helpers
compress/Zlib              zlib compression helpers
csv/Csv                    CSV parse/generate helpers
digest/Digest              digest/hash helpers
dir/Dir                    directory helpers
file/File                  file helpers
geometry/Circle            circle value
geometry/Point             point value
geometry/Rect              rectangle value
geometry/Size              size value
geometry/Vector2           2D vector value
geometry/Vector3           3D vector value
hex/Hex                    hexadecimal encode/decode helpers
image/Codec                image codec helpers
image/Image                image value
io/Io                      stream helpers
io/Reader                  readable stream wrapper
io/Writer                  writable stream wrapper
json/Json                  JSON parse/generate helpers
log/Logger                 logger
markdown/Markdown          Markdown renderer
math/Math                  numeric helpers
matrix/Matrix              matrix value
net/http/Client            HTTP client
net/http/Next              HTTP middleware continuation
net/http/Server            HTTP router/server
net/ip/Address             IP address value
net/ip/Network             IP network value
net/socket/Server          socket listener
net/socket/Socket          socket connection
os/Os                      operating-system helpers
path/Path                  path manipulation helpers
process/Process            process helpers
random/Random              random helpers
random/Rng                 seeded random generator
runtime/Runtime            runtime introspection helpers
secure_random/SecureRandom secure random helpers
serialization/Serializer   data serialization helpers
sync/AtomicInteger         native atomic integer
sync/Mutex                 native mutex
sync/WaitGroup             native wait group
task/Task                  task helpers
template/Template          template renderer
time/Time                  time value and time helpers
toml/Toml                  TOML parse/generate helpers
transform2d/Transform2D    2D affine transform value
unittest/TestCase          test case base class
unittest/TestRunner        test runner
unittest/TestSuite         test suite
url/Url                    URL parse/build helpers
value/Value                value introspection helpers
xml/Xml                    XML parse/generate helpers
xml/Document               XML document node
xml/Element                XML element node
xml/Text                   XML text node
xml/CData                  XML CDATA node
xml/Comment                XML comment node
```

現在の標準ライブラリ protocol と sequence helper files:

```text
comparable                 Comparable interface
equatable                  Equatable interface
stringable                 Stringable interface
iterator                   Iterator interface; requires has_next() and next()
iterable                   Iterable interface; requires iter()
sequence                   Sequence class and chainable sequence protocol
iterable_sequence          sequence wrapper for Iterable values
map_sequence               lazy map sequence
filter_sequence            lazy filter sequence
take_sequence              lazy take sequence
drop_sequence              lazy drop sequence
```

`Comparable` は `compare(other)` を要求し、`lt?`、`lte?`、`gt?`、`gte?`、`between?` を提供する。`Equatable` は `equal?(other)` を要求する。`Stringable` は `to_s()` を要求する。`Iterable` は `iter()` を要求し、`sequence()` を提供する。`Sequence` は `iter()`、`map(fn)`、`filter(fn)`、`take(n)`、`drop(n)`、`reduce(initial, fn)`、`to_a()` を提供する。

`io/Reader`、`io/Writer`、`net/socket` は readable、writable、closable、flushable values の stream capability interfaces を定義する。`Reader` は `read`、`read_line`、`each_line`、`eof?`、`close` をサポートする。`Writer` は `write`、`write_line`、`flush`、`close` をサポートする。`Socket` は `connect`、`read`、`read_line`、`write`、`write_line`、`close`、`closed?`、`local_address`、`remote_address` をサポートする。`net/socket/Server` は `listen`、`accept`、`close`、`local_address` をサポートする。compiled runtime は POSIX socket platform と Windows WinSock2 で `net/socket` をサポートする。

`net/http/Server` は HTTP method ごとの route registration (`get`, `post`, `put`, `delete`, `patch`, `options`, `head`, `any`)、middleware (`use`, `group`)、error と not-found handlers、static-file serving、redirect、route dispatch、server execution を定義する。`net/http/Client` は `get`、`post`、generic `request` を定義する。

`net/http/Client` は `http://` と `https://` URL を受け付ける。HTTPS は compiled runtime の OpenSSL backend を使う。certificate verification は default で有効で、request options は PEM CA bundle を指定する `ca_file`、または明示的に verification を無効化する `insecure_skip_verify: true` を受け付ける。TLS failure は `http.tls:` または `http.request:` error として raise される。`net/http/Server.run_tls(port, cert_file, key_file, options)` は PEM certificate と private key file を使って HTTPS を serve する。options は `host` と `timeout` を受け付ける。TLS-enabled program の build には OpenSSL headers と libraries が必要になる。

compiled `net/http/Server` handler が受け取る request dictionary は、incoming `Cookie` header から parse された `cookies` dictionary を持つ。cookie がない場合は `{}` になる。`=` のない malformed pair は無視され、name と value 周辺の whitespace は trim され、同じ name が複数回出た場合は最後の value が残る。
handler は `form` と `files` dictionary も受け取る。non-multipart request ではどちらも空になる。`multipart/form-data` request では、`form` は field name から最後の string value への mapping、`files` は field name から最後の uploaded file metadata dictionary への mapping になる。file metadata は `filename`、`content_type`、bytes の `body`、`size` を含む。元の raw request body は `body` に残る。malformed multipart body は handler 実行前に `400 Bad Request` を返す。

`Server.cookie(name, value, options)` は `Set-Cookie` header value を format する。options は `path`、`domain`、`max_age`、`expires`、`secure`、`http_only`、`same_site` (`Lax`、`Strict`、`None`) を受け付ける。`SameSite=None` は `secure: true` を要求する。`Server.with_cookie(response, name, value, options)` は `response["header_values"]["Set-Cookie"]` に cookie を追加する。response dictionary は repeated response header のために `header_values` を使える。各 array entry は個別の header line として出力され、通常の `headers` 動作は変わらない。

`Server.render(template, data, options)` と `Server.render_html(template, data, options)` は rendered HTML body を持つ response dictionary を返す。`options` は `nil` にできる。default response status は `200`、default `Content-Type` は `text/html; charset=utf-8`。options は `status`、`headers`、`content_type`、`template_options` を受け付ける。既存 file を指す string template は `template.Template.render_file` で render され、それ以外の string は template source として render される。embedded bytes は render 前に UTF-8 text に decode される。`render_html` は `template_options` がある場合でも HTML escaping を強制する。追加 headers は default の後に merge されるため、caller は `Content-Type` を override できる。

response dictionary は `chunked: true` によって HTTP/1.1 chunked response を送れる。この mode では runtime が `Transfer-Encoding: chunked` を書き、`Content-Length` を省略し、array body の各 string または bytes item を 1 chunk として書く。channel body は string または bytes chunk を yield でき、channel close で stream を閉じる。空 chunk は final terminating chunk を除いて skip される。non-chunked response は通常の `Content-Length` 動作を保つ。

HTTP/1.1 server connection は request に `Connection: close` がない限り keep-alive になる。HTTP/1.0 connection は request に `Connection: keep-alive` がない限り close になる。request dictionary はその request の判定を boolean の `keep_alive` として公開する。response は `Connection: keep-alive` または `Connection: close` を含み、各 accepted connection は保守的な最大 request 数で制限される。

`serialization/Serializer` は Tya values を data values、JSON、TOML と相互変換する。`Serializable` を実装する class は `to_data()` を公開する。

## 外部パッケージ

外部パッケージとツールはこのリポジトリの外で配布され、`tya.toml` を通して git URL と tag、branch、または revision によって消費される。

既知の公開パッケージとツールは次を含む。

- `https://github.com/komagata/tya-sqlite`
- `https://github.com/komagata/tya-sdl2`
- `https://github.com/komagata/tya-gtk4`
- `https://github.com/komagata/tya-raylib`
- `https://github.com/komagata/tya-slim`
- `https://github.com/komagata/flakewatch`
- `https://github.com/komagata/magvideo`

## システム上の考慮事項

Tya プログラムは C にコンパイルされ、Tya ランタイムへリンクされる。ランタイムは、値表現、garbage collection、primitive methods、class dispatch、task と channel support、resources、native wrapper integration を提供する。

実装は `ROADMAP.md` に文書化された self-host fixed-point invariant を保持しなければならない。`selfhost/v01/` の下で保守される Tya 製コンパイラは、自分自身を安定した stage-2 と stage-3 output へコンパイルし続けなければならない。

compiler frontend は手書きである。parser generators と大きな grammar frameworks は、active compiler path に対する言語 authority ではない。
