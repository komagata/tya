module.exports = grammar({
  name: "tya",

  extras: ($) => [/[ \t]/, $.comment],

  word: ($) => $.identifier,

  rules: {
    source_file: ($) => repeat($._statement),

    _statement: ($) =>
      choice(
        $.import_statement,
        $.module_declaration,
        $.class_declaration,
        $.interface_declaration,
        $.member_declaration,
        $.function_assignment,
        $.assignment,
        $.control_statement,
        $.expression_statement,
        $.line_statement,
        "\n",
      ),

    import_statement: ($) =>
      seq("import", $.qualified_identifier, optional(seq("as", $.identifier))),

    module_declaration: ($) => seq("module", field("name", $.identifier)),

    class_declaration: ($) =>
      seq(
        optional(choice("abstract", "final")),
        "class",
        field("name", $.type_identifier),
        optional(seq("extends", $.type_identifier)),
        optional(seq("implements", commaSep1($.type_identifier))),
      ),

    interface_declaration: ($) =>
      seq(
        "interface",
        field("name", $.type_identifier),
        optional(seq("extends", commaSep1($.type_identifier))),
      ),

    function_assignment: ($) =>
      prec.right(2, seq(field("name", $.identifier), "=", $.lambda, optional($.line_tail))),

    member_declaration: ($) =>
      prec.right(
        2,
        seq(
          optional(choice("private", "static", seq("private", "static"), "abstract", "override")),
          field("name", $.identifier),
          ":",
          choice($.lambda, $._expression),
          optional($.line_tail),
        ),
      ),

    assignment: ($) =>
      prec.right(1, seq(field("target", choice($.identifier, $.member)), "=", $._expression, optional($.line_tail))),

    lambda: ($) => prec(2, seq(optional($.parameter_list), "->")),

    parameter_list: ($) =>
      prec(2, choice($.identifier, seq("(", optional(commaSep1($.identifier)), ")"))),

    control_statement: ($) => prec.right(seq($.control_keyword, optional($.line_tail))),

    expression_statement: ($) => prec.right(seq($._expression, optional($.line_tail))),

    line_statement: () => token(/[^\n]+/),

    line_tail: () => token(/[^\n]+/),

    _expression: ($) =>
      choice(
        $.identifier,
        $.self,
        $.super,
        $.boolean,
        $.nil,
        $.number,
        $.string,
        $.bytes,
        $.member,
        $.call,
        $.operator,
      ),

    call: ($) => seq(choice($.identifier, $.member), "(", optional(commaSep1($._expression)), ")"),

    member: ($) => seq(choice($.identifier, $.self, $.super, $.call), repeat1(seq(".", $.identifier))),

    control_keyword: () =>
      choice(
        "if",
        "elseif",
        "else",
        "while",
        "for",
        "in",
        "break",
        "continue",
        "return",
        "raise",
        "try",
        "catch",
        "match",
        "case",
        "when",
        "select",
        "receive",
        "send",
        "timeout",
        "default",
        "spawn",
        "await",
        "scope",
      ),

    self: () => choice("self", "Self"),
    super: () => "super",
    boolean: () => choice("true", "false"),
    nil: () => "nil",

    operator: () =>
      choice(
        "and",
        "or",
        "not",
        "->",
        "==",
        "!=",
        "<=",
        ">=",
        "<<",
        ">>",
        "+",
        "-",
        "*",
        "/",
        "%",
        "=",
        "<",
        ">",
        "&",
        "|",
        "^",
        "~",
      ),

    string: () =>
      choice(
        seq('"', repeat(choice(token.immediate(/[^"\\{}]+/), /\\./, /\{[^}]*\}/)), '"'),
        seq(/[a-z][a-z0-9_]*/, '"""', repeat(/[^"]|("[^"])|(""[^"])/), '"""'),
        seq('"""', repeat(/[^"]|("[^"])|(""[^"])/), '"""'),
        seq(/[a-z][a-z0-9_]*/, /<<<[A-Z][A-Z0-9_]*/, repeat(/[^\n]|\n/), /[ \t]*[A-Z][A-Z0-9_]*/),
        seq(/r<<<[A-Z][A-Z0-9_]*/, repeat(/[^\n]|\n/), /[ \t]*[A-Z][A-Z0-9_]*/),
      ),

    bytes: () =>
      choice(
        seq('b"', repeat(choice(token.immediate(/[^"\\]+/), /\\./)), '"'),
        seq(/b<<<[A-Z][A-Z0-9_]*/, repeat(/[^\n]|\n/), /[ \t]*[A-Z][A-Z0-9_]*/),
      ),

    number: () =>
      token(
        choice(
          /0x[0-9a-fA-F_]+/,
          /0b[01_]+/,
          /[0-9][0-9_]*(\.[0-9][0-9_]*)?/,
        ),
      ),

    qualified_identifier: ($) => seq($.identifier, repeat(seq(".", $.identifier))),
    identifier: () => /[A-Za-z_][A-Za-z0-9_?]*/,
    type_identifier: () => /[A-Z][A-Za-z0-9_]*/,
    comment: () => token(seq("#", /.*/)),
  },
});

function commaSep1(rule) {
  return seq(rule, repeat(seq(",", rule)));
}
