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

この文書は言語を規定する。組み込み関数は `docs/API.md` に列挙されている。標準ライブラリのパッケージと API は `docs/STDLIB.md` に列挙されている。再利用可能なユーザーライブラリとパッケージは `docs/LIBRARIES.md` に記述されている。正規整形の詳細は `docs/CANONICAL_SYNTAX.md` に記述されている。

## 表記

例は通常の Tya ソースを使う。文法断片は説明用であり、完全なパーサ文法ではない。例の名前は `docs/NAMING.md` に従う。

```text
snake_case            変数、関数、メソッド、インポートパスセグメント
_snake_case           インポート可能ファイル内のファイルプライベートなトップレベル束縛
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
extends false final for if implements import in interface module nil not of or
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

Tya は暗黙変換を行わない。数値、文字列、配列、辞書、関数、クラス、タスク、チャンネル、リソースを必要とする操作は、必要な kind の値を受け取らなければならず、そうでなければ実行時エラーを raise する。

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

小文字の `.tya` ファイルはスクリプトファイルである。`tya run` のエントリファイルにでき、直接インポートすることもできる。インポートされた場合、その公開トップレベル束縛はインポート束縛を通して公開される。

`PascalCase` の `.tya` ファイルはクラスファイルである。ライブラリ専用であり、エントリファイルにはできない。クラスファイルは、`.tya` を除いたファイル名と一致する名前の公開クラスをちょうど 1 つ宣言しなければならない。

クラスファイルは、ディレクトリパッケージの一部として明示的に読み込まれる場合も、エントリスクリプトと同じディレクトリにある兄弟として暗黙に読み込まれる場合もある。スクリプトエントリは、同じディレクトリの `PascalCase` クラスファイルをインポートなしで見る。

## 正規構文

Tya には正規構文がある。整形式のすべてのプログラムは 1 つのソース表現を持つ。そのため `tya format` は任意のスタイルツールではなく、言語表面の一部である。

正規構文は、インデント、空行、コメント付着、インポートグループ化、演算子まわりの空白、単一行形式と複数行形式、文字列リテラル形式、空コレクション形式、その他のソース形状の判断を扱う。完全な規則は `docs/CANONICAL_SYNTAX.md` にある。

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

`_` で始まる名前は、インポート可能なソースファイル内のトップレベル束縛である場合に private である。private なトップレベル束縛はファイルインポート束縛を通してエクスポートされない。

定数は `SCREAMING_SNAKE_CASE` を使い、命名規則と代入規則によって定数として検査される。

現在の Tya では、クラスメンバーの private 性に `_` 接頭辞を使わない。private なクラスフィールド、メソッド、クラス変数、クラスメソッド、コンストラクタには `private` キーワードを使う。`_` で始まるクラスメンバー名は不正であり、リネームするか `private` で印を付けるべきである。

```tya
class User
  private id = 0

  private normalize = ->
    self.id.to_s()
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
- `static` クラスメソッドとクラス変数
- `abstract class` と抽象メソッド
- `final class`
- 明示的なメソッド override 検査のための `override`
- `.class` による実行時クラス検査
- ランタイムで文書化された `class`、`class_name`、`name`、`parent` などの読み取り専用クラスメタデータメンバー

```tya
class Admin extends User
  initialize = name ->
    super(name)

  override label = ->
    "admin:{self.name}"
```

クラスファイルは `PascalCase` の `.tya` ファイルである。ファイル名と一致する名前の公開クラスをちょうど 1 つ宣言しなければならない。private な補助クラスとインターフェイスも宣言できる。クラスファイルはライブラリファイルであり、エントリスクリプトとして実行できない。

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
    self.created_at = Time.now()

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

算術演算は、文書化されたプリミティブメソッドまたは演算子ケースが別のことを述べる場合を除き、数値を要求する。`nil` 算術は不正である。

ビット演算子は整数互換の数値を要求する。

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

チャンネルと sync リソースは、`docs/STDLIB.md` に文書化されたメソッドを持つ、標準ライブラリに支えられたランタイム値である。

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
logger.info("started")
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

`for ... of` は辞書の key と value に対する互換綴りとして残るが、新しいコードは意図に応じて `for entry in dict`、`dict.keys()`、`dict.values()` を優先すべきである。

```tya
for key, value of user
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

正確なチャンネルメソッドと sync primitive は `docs/STDLIB.md` で定義される。

## 組み込み関数

Tya には、中核ランタイム操作、I/O、変換、エラー、プロセスアクセス、ファイル、コレクション、コンパイラ introspection のための事前宣言された組み込みがある。規範的な一覧は `docs/API.md` である。

よく使う例:

```tya
print("hello")
args()
type(value)
error("message")
read_file("memo.txt")
write_file("memo.txt", "text")
```

標準ライブラリ API は、ユーザーコードと同じ `import` 構文でインポートされる。

## インポートとパッケージ

### インポート構文

インポートはトップレベルで、他の宣言や文より前に現れる。

```tya
import greeting
import net/http
import net/http as http
```

インポートパスはスラッシュ区切りの `snake_case` セグメントである。相対ファイルシステムパス、絶対パス、空セグメント、終端が `PascalCase` のセグメントは不正である。

### 単一ファイルインポート

単一ファイルインポートは、インポートパスを小文字の `.tya` ファイルへ解決する。

```text
import greeting          -> greeting.tya
import http/server       -> http/server.tya
```

インポートされたファイルは、公開トップレベル束縛をインポート束縛を通して公開する。

```tya
import greeting

print(greeting.hello("komagata"))
```

### ディレクトリパッケージ

ディレクトリパッケージは、インポートパスによって解決される、`PascalCase` クラスファイルを含むディレクトリである。少なくとも 1 つのクラスファイルを含まなければならず、パッケージの leaf に小文字スクリプトファイルを含んではならない。

alias なしのディレクトリインポートは、公開クラス名とインターフェイス名を直接公開する。

```tya
import net/http

server = Server()
```

alias 付きディレクトリインポートは namespace 束縛を公開し、公開名を裸の名前としてインポートしない。

```tya
import net/http as http

server = http.Server()
```

同じディレクトリパッケージ内では、兄弟の公開クラスがインポートなしの裸の `PascalCase` 名で見える。

### 解決順序

インポートは次の順序で解決される。

1. インポート元ファイルのディレクトリ
2. `tya.lock` の manifest 宣言依存
3. `TYA_PATH` に列挙されたディレクトリ。左から右
4. バンドルされた `stdlib/` ディレクトリ

最初に一致したファイルまたはパッケージディレクトリが採用される。ローカルアプリケーションインポートは、パッケージ依存、`TYA_PATH`、標準ライブラリインポートを shadow してよい。パッケージ依存は `TYA_PATH` と標準ライブラリインポートを shadow してよい。

### パッケージマニフェスト

`tya.toml` はパッケージメタデータ、依存、native wrapper、パッケージ提供ツールを宣言する。`tya install` は依存を解決して `tya.lock` を書く。Git 依存とローカルパス依存がサポートされる。現在、中央パッケージレジストリと `tya publish` コマンドはない。

native パッケージメタデータは `[native]` の下に置かれる。パッケージ提供ツールは `[tools]` の下に置かれ、`tya tool` を通して実行される。

## プログラム実行

スクリプトファイルは小文字の `.tya` ファイルであり、`tya run`、`tya build`、`tya emit-c` のエントリファイルとして使える。

```sh
tya run hello.tya
tya build hello.tya -o hello
tya emit-c hello.tya
```

クラスファイルはライブラリ専用であり、エントリファイルにはできない。

Tya は native 実行のために C へコンパイルするパイプラインを使う。`tya run` は一時 native 実行ファイルをコンパイルし、実行し、その一時実行ファイルを削除する。`tya build` は再利用可能な実行ファイルを書く。`tya emit-c` は Tya ソースから生成された C プログラムを表示または書き出す。生成された C は Tya ランタイムへリンクされる。

デフォルトの native target はホストの C toolchain を使う。`[native]` の native パッケージメタデータは、C ソース、ヘッダー、include ディレクトリ、`pkg-config` フラグ、コンパイラフラグ、リンカフラグをビルドへ寄与する。

WASM build target は、サポートされる場所で利用できる。native パッケージは未サポートの WASM target で拒否される。

cross-compilation は `--target` で選択される。native target がデフォルトである。現在の WebAssembly target は次を含む。

```sh
tya build --target native src/main.tya -o app
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
```

`wasm32-wasi` は WASI `.wasm` artifact を生成する。`wasm32-browser` はブラウザ向け `.wasm` artifact と JavaScript loader を生成する。`tya doctor wasm` は WebAssembly build 環境を報告する。`tya run` は native 専用のままである。

## 組み込みツール

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

`tya run` はスクリプトファイルを一時 native 実行ファイルとしてビルドして実行する。

`tya build` は再利用可能な実行ファイルまたは target artifact をビルドする。サポートされる native target と WebAssembly target 用に `--target` を受け付ける。

`tya emit-c` は、検査または外部ビルドパイプライン用に生成された C プログラムを出力する。

`tya check` はプログラムを実行したり C コンパイラを呼び出したりせず、ソースを parse し、import を load し、検証する。

`tya format` は正規ソースを表示し、`tya format -w` はファイルを in place に書き換える。

`tya test` は標準 `unittest` surface を使ってテストを発見し実行する。`--cover` で coverage data を収集できる。

`tya cover` は coverage profile を人間可読または JSON 形式で報告する。

`tya lint` は安定した linter diagnostics を報告する。autofix と text、JSON、SARIF 出力をサポートする。

`tya lsp` はエディタ統合用に JSON-RPC 上で language server を実行する。

`tya doc` はソースコメントからドキュメントを抽出し、静的 HTML を生成できる。

`tya new` は native package scaffold を含む新規プロジェクトとライブラリを scaffold する。

`tya task` は `tya.toml` で宣言されたタスクを一覧表示および実行する。serial と parallel の task 形式を含む。

`tya install` は依存を解決して `tya.lock` を書く。

`tya update` は lock された依存バージョンを更新する。

`tya add` は git、tag、revision、branch、path、dev dependencies を含む manifest dependencies を追加する。

`tya remove` は manifest dependencies を削除する。

`tya outdated` はより新しい利用可能バージョンがある依存を報告する。

`tya tool` はパッケージ提供ツールを一覧表示および実行する。ローカルパスまたは pin された git source からの one-shot tool execution もサポートする。

`tya doctor` は native と WebAssembly build のための host environment details を報告する。

`tya embed` は embedded asset handling を検査または支援する。

`tya version` はインストールされた Tya バージョンを表示する。

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

標準ライブラリは `stdlib/` の下で Tya に同梱され、ユーザーファイルやパッケージと同じ import 構文でインポートされる。

例:

```text
math
path
file
json
toml
csv
url
time
random
unittest
template
markdown
compress
log
io
net/ip
net/socket
net/http
channel
sync
task
```

規範的な標準ライブラリ API reference は `docs/STDLIB.md` である。

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
