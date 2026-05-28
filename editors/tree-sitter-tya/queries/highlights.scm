[
  "if"
  "elseif"
  "else"
  "while"
  "for"
  "in"
  "break"
  "continue"
  "return"
  "raise"
  "try"
  "catch"
  "match"
  "case"
  "when"
  "select"
  "receive"
  "send"
  "timeout"
  "default"
] @keyword.control

[
  "class"
  "module"
  "interface"
  "struct"
  "record"
  "implements"
  "extends"
  "abstract"
  "final"
  "private"
  "static"
  "initialize"
  "import"
  "as"
] @keyword

[
  "spawn"
  "await"
  "scope"
] @keyword.coroutine

[
  "and"
  "or"
  "not"
] @operator

(comment) @comment
(string) @string
(bytes) @string.special
(number) @number
(boolean) @boolean
(nil) @constant.builtin
(self) @variable.builtin
(super) @variable.builtin
(type_identifier) @type

(class_declaration name: (type_identifier) @type.definition)
(interface_declaration name: (type_identifier) @type.definition)
(module_declaration name: (identifier) @namespace)
(function_assignment name: (identifier) @function)
(call (identifier) @function.call)
