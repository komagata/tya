---
layout: doc
title: ガイド
lang: ja
permalink: /ja/guide/
language_url: /guide/
---

# Tya ガイド

このガイドは、Tya を初めて触る人向けです。インストールから始めて、小さな
プログラムを書き、実行し、実行ファイルとしてビルドし、基本的な言語機能を
順に試します。

厳密な言語仕様は [仕様](/ja/spec/) を参照してください。このガイドは実践用
です。例をコピーして実行し、少しずつ変更しながら読み進めてください。

## インストール

GitHub の最新リリースから、自分の環境向けの Tya をインストールします。

```sh
curl -fsSL https://tya-lang.org/install.sh | sh
```

手動で入れる場合は、リリースページからダウンロードします。

```text
https://github.com/komagata/tya/releases/latest
```

`tya` コマンドが使えることを確認します。

```sh
tya version
```

Tya はネイティブ実行ファイルをビルドするために、同梱ツールチェイン支援を
使います。ビルドが失敗する場合は次を実行します。

```sh
tya doctor
```

## プログラムを作る

新しいディレクトリを作り、`hello.tya` を作成します。

```sh
mkdir hello-tya
cd hello-tya
```

```tya
name = "Tya"
print("Hello, {name}")
```

実行します。

```sh
tya run hello.tya
```

出力は次のようになります。

```text
Hello, Tya
```

Tya のソースファイルは `.tya` を使います。`hello.tya` や `main.tya` の
ような小文字ファイルは、スクリプトのエントリポイントとして使えます。

## 検査と整形

実行やビルドの前に素早く妥当性を確認したいときは `tya check` を使います。

```sh
tya check hello.tya
```

`tya format` は正規化されたソースを表示します。

```sh
tya format hello.tya
```

`-w` を付けるとファイルを書き換えます。

```sh
tya format -w hello.tya
```

整形は Tya の言語設計の一部です。妥当なプログラムには、正規のソース表現が
1 つだけあります。

## 実行ファイルをビルドする

ネイティブ実行ファイルをビルドします。

```sh
tya build hello.tya -o hello
```

ビルドしたプログラムを実行します。

```sh
./hello
```

Windows では `.exe` の出力名を使います。

```sh
tya build hello.tya -o hello.exe
hello.exe
```

`tya run` は手元で素早く実行するためのコマンドです。`tya build` は再利用
できる実行ファイルを作ります。

## 小さなスクリプト

`hello.tya` を少し大きなプログラムに置き換えます。

```tya
greet = name ->
  "Hello, {name}"

names = ["Ada", "Matz", "Tya"]

for name in names
  print(greet(name))
```

実行します。

```sh
tya run hello.tya
```

この例では 3 つの基本を使っています。

- 関数は値で、`->` を使って書く
- 配列は `[...]` を使う
- インデントがブロックを定義する

## 値

Tya は動的型付けです。値は実行時の kind を持ちます。

```tya
name = "Tya"
count = 3
price = 12.5
ready = true
missing = nil
data = b"abc"
```

文字列では補間が使えます。

```tya
print("count = {count}")
```

配列は変更できます。

```tya
items = [1, 2]
items.push(3)
print(items[0])
print(items.len())
```

辞書は文字列キーを使います。

```tya
user = { name: "Ada", admin: true }
print(user["name"])
user["city"] = "London"
```

プリミティブ値にもメソッドがあります。

```tya
print(" tya ".trim().upper())
print(42.to_string())
print([1, 2, 3].len())
```

## 名前

値、関数、メソッド、ファイル、インポートパス、辞書キーには `snake_case` を
使います。

クラスとインターフェイスには `PascalCase` を使います。

定数には `SCREAMING_SNAKE_CASE` を使います。

```tya
user_name = "Ada"
MAX_RETRIES = 3

class UserProfile
  initialize = name ->
    self.name = name
```

先頭の `_` は private を意味しません。クラスメンバーを private にするには
`private` を使います。

## 制御構文

条件分岐には `if`、`elseif`、`else` を使います。

```tya
age = 20

if age >= 20
  print("adult")
elseif age >= 13
  print("teen")
else
  print("young")
```

真偽値の組み合わせには `and`、`or`、`not` を使います。

```tya
if ready and not disabled
  print("ready")
```

繰り返しには `while` を使えます。

```tya
count = 0
while count < 3
  print(count)
  count = count + 1
```

配列の反復には `for` を使います。

```tya
items = ["a", "b", "c"]

for item in items
  print(item)

for item, index in items
  print("{index}: {item}")
```

`break` は一番近いループを抜けます。`continue` は次の反復に進みます。

## 関数

関数は `->` で書きます。呼び出しでは常に括弧を使います。

```tya
add = a, b -> a + b
print(add(2, 3))
```

複数の文を書く場合はインデントした本体を使います。最後の式が返り値です。

```tya
double = value ->
  result = value * 2
  result

print(double(21))
```

早く返したい場合は `return` を使います。

```tya
label = name ->
  if name == ""
    return "anonymous"
  name
```

関数は他の関数に渡せます。

```tya
items = [1, 2, 3]
print(items.map(item -> item * 2))
```

## エラー

`error(...)` でエラー値を作ります。失敗を扱うには `raise`、`try`、`catch`
を使います。

```tya
require_name = name ->
  if name == ""
    raise error("name is required")
  name

try
  print(require_name(""))
catch err
  message = err["message"]
  print("error: {message}")
```

後始末を必ず実行したい場合は `try/finally` を使います。

```tya
try
  print("work")
finally
  print("cleanup")
```

## ファイルに分ける

`greeting.tya` を作ります。

```tya
hello = name ->
  "Hello, {name}"
```

`main.tya` を作ります。

```tya
import greeting

print(greeting.hello("Tya"))
```

エントリファイルを実行します。

```sh
tya run main.tya
```

パッケージ名が長い場合は alias を使えます。

```tya
import net/http/* as http

resp = http.Client().get("https://example.com/")
print(resp["status"])
```

## クラス

クラスは実行時の値です。`initialize` はコンストラクタの hook です。

```tya
class User
  initialize = name ->
    self.name = name

  label = ->
    "user:{self.name}"

user = User("Ada")
print(user.label())
```

private なクラスメンバーには `private` を使います。

```tya
class Counter
  private count = 0

  increment = ->
    self.count = self.count + 1
    self.count
```

クラスファイルは `user.tya` のような snake_case ファイル名を使い、中では `class User` のような PascalCase クラスを宣言します。ライブラリ用のファイルであり、スクリプトのエントリポイントではありません。

## 標準ライブラリ

標準ライブラリのパッケージは、ユーザーファイルと同じように import します。

```tya
import json
import time

json_text = json.Json().dump({ name: "Tya" })
print(json_text)

now = time.Time().now()
print(now.format("rfc3339"))
```

よく使うパッケージには `json`、`toml`、`csv`、`url`、`time`、`random`、
`math`、`file`、`path`、`unittest`、`template`、`markdown`、`compress`、
`log`、`io`、`net/ip`、`net/socket`、`net/http`、`channel`、`sync` などが
あります。

ソースコメントから API ドキュメントを生成できます。

```sh
tya doc --json lib
```

## テスト

Tya のテストファイルは `_test.tya` で終わります。

`hello_test.tya` を作ります。

```tya
import unittest

class AdditionTest extends TestCase
  test_addition_works = ->
    self.assert_equal(4, 2 + 2, "addition")
```

テストを実行します。

```sh
tya test
```

coverage 対応のテストを使う場合は coverage を表示できます。

```sh
tya test --cover
tya cover
```

## Lint とドキュメント

lint を実行します。

```sh
tya lint
tya lint --fix .
```

ソースコメントからドキュメントを生成します。

```sh
tya doc
tya doc --html ./site src
```

## パッケージ

プロジェクトのメタデータと依存は `tya.toml` に書きます。解決済み依存は
`tya.lock` に書かれます。

```sh
tya install
tya add https://github.com/komagata/tya-sqlite --tag v0.1.0
tya update
tya outdated
```

Git 依存とローカルパス依存が使えます。現在、Tya は中央パッケージレジストリ
を使いません。

## クロスコンパイル

ネイティブターゲットを明示してビルドします。

```sh
tya build --target native main.tya -o app
```

WebAssembly ターゲットをビルドします。

```sh
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
tya doctor wasm
```

`tya run` は native 専用です。

## よく使うコマンド

```sh
tya version
tya run main.tya
tya check main.tya
tya format -w .
tya build main.tya -o app
tya test
tya lint
tya doc
tya doctor
```

このガイドの次は、正確な構文、実行時の挙動、パッケージ規則、標準ライブラリ
の境界が必要になったときに [仕様](/ja/spec/) を読んでください。
