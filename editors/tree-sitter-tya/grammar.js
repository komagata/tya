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
        $.function_assignment,
        $.control_keyword,
        $.expression_statement,
        "\n",
      ),

    import_statement: ($) =>
      seq("import", $.identifier, optional(seq("as", $.identifier))),

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
      prec(1, seq(field("name", $.identifier), "=", $.lambda)),

    lambda: ($) => seq(optional($.parameter_list), "->"),

    parameter_list: ($) =>
      choice($.identifier, seq("(", optional(commaSep1($.identifier)), ")")),

    expression_statement: ($) => $._expression,

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
        $.call,
        $.member,
        $.operator,
      ),

    call: ($) => seq($.identifier, "(", optional(commaSep1($._expression)), ")"),

    member: ($) => seq($._expression, ".", $.identifier),

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
        seq('"""', repeat(/[^"]|("[^"])|(""[^"])/), '"""'),
      ),

    bytes: () => seq('b"', repeat(choice(token.immediate(/[^"\\]+/), /\\./)), '"'),

    number: () =>
      token(
        choice(
          /0x[0-9a-fA-F_]+/,
          /0b[01_]+/,
          /[0-9][0-9_]*(\.[0-9][0-9_]*)?/,
        ),
      ),

    identifier: () => /[A-Za-z_][A-Za-z0-9_?]*/,
    type_identifier: () => /[A-Z][A-Za-z0-9_]*/,
    comment: () => token(seq("#", /.*/)),
  },
});

function commaSep1(rule) {
  return seq(rule, repeat(seq(",", rule)));
}
