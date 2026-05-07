# Tya v0.1 API

この文書は Tya v0.1 の標準組み込み関数を定義する。

v0.1 では、self-host compiler と基本的なプログラムに必要な最小 API だけを
標準組み込みとして固定する。便利関数は後続バージョンで追加する。

## Core

```tya
print value
panic "bad state"
exit 1
```

`print` は値と newline を出力する。`panic` はエラーとして停止する。`exit` は
status code を指定して終了する。

```tya
err = error "file not found"
print err["message"]
```

`error` は `message` を持つ error value を返す。v0.1 では `.` が module
member access 専用なので、message は `err["message"]` で読む。

## Conversion

```tya
to_string value
to_int value
to_float value
to_number value
```

```tya
print to_string 20
print to_int "42"
print to_float "2.5"
print to_number "12.5"
```

## Strings

```tya
split text, separator
join items, separator
trim text
replace text, old, new
contains text, search
starts_with text, prefix
ends_with text, suffix
```

```tya
text = trim "  hello,tya  "
parts = split text, ","

print join parts, "-"
print replace text, "tya", "Tya"
print contains text, "hello"
print starts_with text, "hello"
print ends_with text, "tya"
```

## Arrays

```tya
len value
push array, value
pop array
```

```tya
items = [1, 2]
push items, 3
print pop items
print len items
```

`len` は string、array、dictionary に使える。

## Dictionaries

```tya
keys dictionary
values dictionary
has dictionary, key
delete dictionary, key
```

```tya
user = { name: "komagata", age: 20 }

print keys user
print values user
print has user, "name"
delete user, "age"
```

## Files

```tya
read_file path
write_file path, text
file_exists path
```

```tya
write_file "/tmp/memo.txt", "hello"
print read_file "/tmp/memo.txt"
print file_exists "/tmp/memo.txt"
```

## Process

```tya
args()
env name
```

```tya
items = args()
print len items
print env "HOME"
```

## Not In v0.1

以下は v0.1 標準組み込みに含めない。

```text
map
filter
find
any
all
each
reduce
byte_len
char_len
equal
div
read_line
set
```

## Naming

標準組み込み関数は `snake_case` を使う。CamelCase の builtin spelling は
言語仕様に含めない。
