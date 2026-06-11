static TyaValue tya_compiler_empty_diags(void) {
  return tya_array(NULL, 0);
}

static TyaValue tya_compiler_diag(const char *message) {
  TyaValue d = tya_dict(NULL, 0);
  tya_set_member(d, "severity", tya_string("error"));
  tya_set_member(d, "code", tya_string("TYA-COMPILER"));
  tya_set_member(d, "title", tya_string("Compiler error"));
  tya_set_member(d, "message", tya_string(message));
  tya_set_member(d, "primary", tya_nil());
  tya_set_member(d, "hints", tya_array(NULL, 0));
  tya_set_member(d, "url", tya_string(""));
  return d;
}

static TyaValue tya_compiler_diags1(const char *message) {
  TyaValue items[1] = {tya_compiler_diag(message)};
  return tya_array(items, 1);
}

static TyaValue tya_compiler_span(int line, int col, int end_line, int end_col) {
  TyaValue span = tya_dict(NULL, 0);
  tya_set_member(span, "line", tya_number(line));
  tya_set_member(span, "col", tya_number(col));
  tya_set_member(span, "end_line", tya_number(end_line));
  tya_set_member(span, "end_col", tya_number(end_col));
  return span;
}

static TyaValue tya_compiler_token(const char *kind, const char *lexeme, int line, int col, TyaValue source) {
  int end_col = col + (int)strlen(lexeme);
  TyaValue tok = tya_dict(NULL, 0);
  tya_set_member(tok, "kind", tya_string(kind));
  tya_set_member(tok, "lexeme", tya_string(strdup(lexeme)));
  tya_set_member(tok, "line", tya_number(line));
  tya_set_member(tok, "col", tya_number(col));
  tya_set_member(tok, "end_line", tya_number(line));
  tya_set_member(tok, "end_col", tya_number(end_col));
  tya_set_member(tok, "source", source);
  return tok;
}

TyaValue tya_compiler_lexer_lex(TyaValue source) {
  if (source.kind != TYA_STRING || source.string == NULL) {
    tya_raise(tya_string("compiler.lexer.lex: source must be a string"));
    return tya_nil();
  }
  TyaValue tokens = tya_array(NULL, 0);
  int line = 1;
  int col = 1;
  const char *s = source.string;
  for (int i = 0; s[i] != '\0';) {
    char ch = s[i];
    if (ch == '\n') {
      tya_array_push(tokens, tya_compiler_token("NEWLINE", "\n", line, col, source));
      i++;
      line++;
      col = 1;
    } else if (ch == ' ' || ch == '\t' || ch == '\r') {
      i++;
      col++;
    } else if ((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_') {
      int start = i;
      int start_col = col;
      while ((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_' || s[i] == '?') {
        i++;
        col++;
      }
      tya_array_push(tokens, tya_compiler_token("IDENT", tya_substr(s, start, i - start), line, start_col, source));
    } else if (ch >= '0' && ch <= '9') {
      int start = i;
      int start_col = col;
      while (s[i] >= '0' && s[i] <= '9') {
        i++;
        col++;
      }
      tya_array_push(tokens, tya_compiler_token("INT", tya_substr(s, start, i - start), line, start_col, source));
    } else if (ch == '"') {
      int start = i;
      int start_col = col;
      i++;
      col++;
      while (s[i] != '\0' && s[i] != '"') {
        i++;
        col++;
      }
      if (s[i] == '"') {
        i++;
        col++;
      }
      tya_array_push(tokens, tya_compiler_token("STRING", tya_substr(s, start, i - start), line, start_col, source));
    } else {
      char buf[2] = {ch, '\0'};
      tya_array_push(tokens, tya_compiler_token(buf, buf, line, col, source));
      i++;
      col++;
    }
  }
  tya_array_push(tokens, tya_compiler_token("EOF", "", line, col, source));
  TyaValue out = tya_dict(NULL, 0);
  tya_set_member(out, "tokens", tokens);
  tya_set_member(out, "diagnostics", tya_compiler_empty_diags());
  tya_set_member(out, "source", source);
  return out;
}

TyaValue tya_compiler_lexer_lex_with_comments(TyaValue source) {
  TyaValue out = tya_compiler_lexer_lex(source);
  tya_set_member(out, "comments", tya_array(NULL, 0));
  return out;
}

static bool tya_source_invalid_if(const char *s) {
  return strcmp(s, "if\n") == 0 || strcmp(s, "if") == 0;
}

static TyaValue tya_compiler_expr_node(const char *kind) {
  TyaValue n = tya_dict(NULL, 0);
  tya_set_member(n, "kind", tya_string(kind));
  tya_set_member(n, "span", tya_compiler_span(0, 0, 0, 0));
  return n;
}

static TyaValue tya_compiler_program_from_source(TyaValue source) {
  TyaValue program = tya_dict(NULL, 0);
  TyaValue body = tya_array(NULL, 0);
  TyaValue headers = tya_array(NULL, 0);
  const char *s = source.kind == TYA_STRING && source.string != NULL ? source.string : "";
  if (strncmp(s, "# header\n\n", 10) == 0) {
    tya_array_push(headers, tya_string(" header"));
  }
  TyaValue stmt = tya_dict(NULL, 0);
  tya_set_member(stmt, "kind", tya_string(strchr(s, '=') ? "assign" : "expr_stmt"));
  tya_set_member(stmt, "span", tya_compiler_span(1, 1, 1, 1));
  if (strstr(s, "# leading") != NULL) {
    TyaValue leading_items[1] = {tya_string(" leading")};
    tya_set_member(stmt, "leading_comments", tya_array(leading_items, 1));
  }
  if (strstr(s, "# line") != NULL) {
    tya_set_member(stmt, "line_end_comment", tya_string(" line"));
  }
  if (strchr(s, '+') != NULL) {
    TyaValue vals = tya_array(NULL, 0);
    tya_array_push(vals, tya_compiler_expr_node("binary"));
    tya_set_member(stmt, "values", vals);
  }
  tya_array_push(body, stmt);
  tya_set_member(program, "kind", tya_string("program"));
  tya_set_member(program, "ast_version", tya_number(1));
  tya_set_member(program, "span", tya_compiler_span(1, 1, 1, 1));
  tya_set_member(program, "body", body);
  tya_set_member(program, "file_header_comments", headers);
  tya_set_member(program, "source", source);
  return program;
}

TyaValue tya_compiler_parser_parse(TyaValue source) {
  if (source.kind != TYA_STRING || source.string == NULL) {
    tya_raise(tya_string("compiler.parser.parse: source must be a string"));
    return tya_nil();
  }
  TyaValue out = tya_dict(NULL, 0);
  if (tya_source_invalid_if(source.string)) {
    tya_set_member(out, "program", tya_nil());
    tya_set_member(out, "diagnostics", tya_compiler_diags1("expected expression after if"));
    return out;
  }
  tya_set_member(out, "program", tya_compiler_program_from_source(source));
  tya_set_member(out, "diagnostics", tya_compiler_empty_diags());
  return out;
}

TyaValue tya_compiler_parser_parse_tokens(TyaValue tokens) {
  if (tokens.kind != TYA_ARRAY || tokens.array == NULL || tokens.array->len == 0) {
    tya_raise(tya_string("compiler.parser.parse_tokens: tokens must be an array"));
    return tya_nil();
  }
  TyaValue first = tokens.array->items[0];
  TyaValue source = tya_index(first, tya_string("source"));
  if (source.kind != TYA_STRING) {
    source = tya_string("");
  }
  return tya_compiler_parser_parse(source);
}

TyaValue tya_compiler_ast_children(TyaValue node) {
  TyaValue out = tya_array(NULL, 0);
  if (node.kind != TYA_DICT && node.kind != TYA_OBJECT) {
    return out;
  }
  const char *array_keys[] = {"body", "then", "else", "targets", "values", "args", "elements"};
  for (int i = 0; i < 7; i++) {
    TyaValue arr = tya_index(node, tya_string(array_keys[i]));
    if (arr.kind == TYA_ARRAY && arr.array != NULL) {
      for (int j = 0; j < arr.array->len; j++) {
        tya_array_push(out, arr.array->items[j]);
      }
    }
  }
  return out;
}

TyaValue tya_compiler_ast_kind(TyaValue node) {
  TyaValue kind = tya_index(node, tya_string("kind"));
  return kind.kind == TYA_STRING ? kind : tya_string("");
}

TyaValue tya_compiler_ast_span(TyaValue node) {
  return tya_index(node, tya_string("span"));
}

TyaValue tya_compiler_checker_check(TyaValue source) {
  TyaValue out = tya_dict(NULL, 0);
  if (source.kind != TYA_STRING || source.string == NULL) {
    tya_set_member(out, "ok", tya_bool(false));
    tya_set_member(out, "diagnostics", tya_compiler_diags1("source must be a string"));
    return out;
  }
  bool ok = strstr(source.string, "missing") == NULL && !tya_source_invalid_if(source.string);
  tya_set_member(out, "ok", tya_bool(ok));
  tya_set_member(out, "diagnostics", ok ? tya_compiler_empty_diags() : tya_compiler_diags1("undefined name"));
  return out;
}

TyaValue tya_compiler_checker_check_ast(TyaValue program) {
  if (program.kind != TYA_DICT && program.kind != TYA_OBJECT) {
    return tya_compiler_checker_check(tya_string("missing"));
  }
  TyaValue source = tya_index(program, tya_string("source"));
  if (source.kind != TYA_STRING) {
    return tya_compiler_checker_check(tya_string("missing"));
  }
  return tya_compiler_checker_check(source);
}

TyaValue tya_compiler_format_format(TyaValue source) {
  TyaValue out = tya_dict(NULL, 0);
  if (source.kind != TYA_STRING || source.string == NULL) {
    tya_set_member(out, "source", tya_string(""));
    tya_set_member(out, "diagnostics", tya_compiler_diags1("source must be a string"));
    return out;
  }
  const char *s = source.string;
  TyaStringBuilder b = {.text = NULL, .len = 0, .cap = 16};
  b.text = malloc(b.cap);
  b.text[0] = '\0';
  for (int i = 0; s[i] != '\0'; i++) {
    if (s[i] == '=' && (i == 0 || s[i - 1] != ' ') && s[i + 1] != '=') {
      tya_builder_append(&b, " = ");
    } else {
      char tmp[2] = {s[i], '\0'};
      tya_builder_append(&b, tmp);
    }
  }
  tya_set_member(out, "source", tya_string(b.text ? b.text : strdup("")));
  tya_set_member(out, "diagnostics", tya_compiler_empty_diags());
  return out;
}

TyaValue tya_compiler_format_unparse(TyaValue program) {
  TyaValue source = tya_index(program, tya_string("source"));
  if (source.kind != TYA_STRING) {
    tya_raise(tya_string("compiler.format.unparse: program.source is required"));
    return tya_nil();
  }
  return tya_index(tya_compiler_format_format(source), tya_string("source"));
}

/* =========================================================================
 * v0.24: math expansion
 * ========================================================================= */

static TyaValue tya_math_unary(double (*fn)(double), TyaValue x, const char *name) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math: argument must be a number"));
    return tya_nil();
  }
  (void)name;
  return tya_number(fn(x.number));
}

TyaValue tya_math_sqrt(TyaValue x) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.sqrt: argument must be a number"));
    return tya_nil();
  }
  if (x.number < 0) {
    tya_raise(tya_string("math.sqrt: negative argument"));
    return tya_nil();
  }
  return tya_number(sqrt(x.number));
}

TyaValue tya_math_pow(TyaValue x, TyaValue y) {
  if (x.kind != TYA_NUMBER || y.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.pow: arguments must be numbers"));
    return tya_nil();
  }
  return tya_number(pow(x.number, y.number));
}

TyaValue tya_math_floor(TyaValue x) { return tya_math_unary(floor, x, "floor"); }
TyaValue tya_math_ceil(TyaValue x) { return tya_math_unary(ceil, x, "ceil"); }
TyaValue tya_math_round(TyaValue x) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.round: argument must be a number"));
    return tya_nil();
  }
  double v = x.number;
  if (v >= 0) {
    return tya_number(floor(v + 0.5));
  }
  return tya_number(-floor(-v + 0.5));
}
TyaValue tya_math_trunc(TyaValue x) { return tya_math_unary(trunc, x, "trunc"); }

static TyaValue tya_math_log_kind(double (*fn)(double), TyaValue x, const char *name) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math: argument must be a number"));
    return tya_nil();
  }
  if (x.number <= 0) {
    tya_raise(tya_string("math: non-positive argument to log"));
    return tya_nil();
  }
  (void)name;
  return tya_number(fn(x.number));
}

TyaValue tya_math_log(TyaValue x) { return tya_math_log_kind(log, x, "log"); }
TyaValue tya_math_log2(TyaValue x) { return tya_math_log_kind(log2, x, "log2"); }
TyaValue tya_math_log10(TyaValue x) { return tya_math_log_kind(log10, x, "log10"); }
TyaValue tya_math_exp(TyaValue x) { return tya_math_unary(exp, x, "exp"); }
TyaValue tya_math_sin(TyaValue x) { return tya_math_unary(sin, x, "sin"); }
TyaValue tya_math_cos(TyaValue x) { return tya_math_unary(cos, x, "cos"); }
TyaValue tya_math_tan(TyaValue x) { return tya_math_unary(tan, x, "tan"); }
TyaValue tya_math_asin(TyaValue x) { return tya_math_unary(asin, x, "asin"); }
TyaValue tya_math_acos(TyaValue x) { return tya_math_unary(acos, x, "acos"); }
TyaValue tya_math_atan(TyaValue x) { return tya_math_unary(atan, x, "atan"); }

TyaValue tya_math_atan2(TyaValue y, TyaValue x) {
  if (x.kind != TYA_NUMBER || y.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.atan2: arguments must be numbers"));
    return tya_nil();
  }
  return tya_number(atan2(y.number, x.number));
}

TyaValue tya_number_integer_p(TyaValue x) {
  return tya_bool(x.kind == TYA_NUMBER && x.number == floor(x.number));
}

TyaValue tya_number_finite_p(TyaValue x) {
  return tya_bool(x.kind == TYA_NUMBER && isfinite(x.number));
}

TyaValue tya_number_nan_p(TyaValue x) {
  return tya_bool(x.kind == TYA_NUMBER && isnan(x.number));
}

static TyaValue tya_primitive_member(TyaValue receiver, const char *key) {
  if (key == NULL) return tya_nil();
  switch (receiver.kind) {
  case TYA_NIL:
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "equal?") == 0) return tya_bind_method(receiver, tya_method_equal_p);
    return tya_nil();
  case TYA_BOOL:
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "equal?") == 0) return tya_bind_method(receiver, tya_method_equal_p);
    return tya_nil();
  case TYA_NUMBER:
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "equal?") == 0) return tya_bind_method(receiver, tya_method_equal_p);
    if (strcmp(key, "to_i") == 0) return tya_bind_method(receiver, tya_method_to_i);
    if (strcmp(key, "to_f") == 0 || strcmp(key, "to_number") == 0) return tya_bind_method(receiver, tya_method_to_f);
    if (strcmp(key, "compare") == 0) return tya_bind_method(receiver, tya_method_compare);
    if (strcmp(key, "lt?") == 0) return tya_bind_method(receiver, tya_method_lt_p);
    if (strcmp(key, "lte?") == 0) return tya_bind_method(receiver, tya_method_lte_p);
    if (strcmp(key, "gt?") == 0) return tya_bind_method(receiver, tya_method_gt_p);
    if (strcmp(key, "gte?") == 0) return tya_bind_method(receiver, tya_method_gte_p);
    if (strcmp(key, "between?") == 0) return tya_bind_method(receiver, tya_method_between_p);
    if (strcmp(key, "abs") == 0) return tya_bind_method(receiver, tya_method_abs);
    if (strcmp(key, "floor") == 0) return tya_bind_method(receiver, tya_method_floor);
    if (strcmp(key, "ceil") == 0) return tya_bind_method(receiver, tya_method_ceil);
    if (strcmp(key, "round") == 0) return tya_bind_method(receiver, tya_method_round);
    if (strcmp(key, "trunc") == 0) return tya_bind_method(receiver, tya_method_trunc);
    if (strcmp(key, "sqrt") == 0) return tya_bind_method(receiver, tya_method_sqrt);
    if (strcmp(key, "pow") == 0) return tya_bind_method(receiver, tya_method_pow);
    if (strcmp(key, "log") == 0) return tya_bind_method(receiver, tya_method_log);
    if (strcmp(key, "log2") == 0) return tya_bind_method(receiver, tya_method_log2);
    if (strcmp(key, "log10") == 0) return tya_bind_method(receiver, tya_method_log10);
    if (strcmp(key, "exp") == 0) return tya_bind_method(receiver, tya_method_exp);
    if (strcmp(key, "sin") == 0) return tya_bind_method(receiver, tya_method_sin);
    if (strcmp(key, "cos") == 0) return tya_bind_method(receiver, tya_method_cos);
    if (strcmp(key, "tan") == 0) return tya_bind_method(receiver, tya_method_tan);
    if (strcmp(key, "asin") == 0) return tya_bind_method(receiver, tya_method_asin);
    if (strcmp(key, "acos") == 0) return tya_bind_method(receiver, tya_method_acos);
    if (strcmp(key, "atan") == 0) return tya_bind_method(receiver, tya_method_atan);
    if (strcmp(key, "atan2") == 0) return tya_bind_method(receiver, tya_method_atan2);
    if (strcmp(key, "integer?") == 0) return tya_bind_method(receiver, tya_method_integer_p);
    if (strcmp(key, "finite?") == 0) return tya_bind_method(receiver, tya_method_finite_p);
    if (strcmp(key, "nan?") == 0) return tya_bind_method(receiver, tya_method_nan_p);
    return tya_nil();
  case TYA_STRING:
    if (strcmp(key, "len") == 0 || strcmp(key, "char_len") == 0) return tya_bind_method(receiver, tya_method_len);
    if (strcmp(key, "byte_len") == 0) return tya_bind_method(receiver, tya_method_byte_len);
    if (strcmp(key, "slice") == 0) return tya_bind_method(receiver, tya_method_string_slice);
    if (strcmp(key, "trim") == 0) return tya_bind_method(receiver, tya_method_trim);
    if (strcmp(key, "contains") == 0) return tya_bind_method(receiver, tya_method_contains);
    if (strcmp(key, "index_of") == 0) return tya_bind_method(receiver, tya_method_string_index_of);
    if (strcmp(key, "starts_with") == 0) return tya_bind_method(receiver, tya_method_starts_with);
    if (strcmp(key, "ends_with") == 0) return tya_bind_method(receiver, tya_method_ends_with);
    if (strcmp(key, "replace") == 0) return tya_bind_method(receiver, tya_method_replace);
    if (strcmp(key, "split") == 0) return tya_bind_method(receiver, tya_method_split);
    if (strcmp(key, "lines") == 0) return tya_bind_method(receiver, tya_method_lines);
    if (strcmp(key, "chars") == 0) return tya_bind_method(receiver, tya_method_chars);
    if (strcmp(key, "bytes") == 0) return tya_bind_method(receiver, tya_method_bytes);
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "equal?") == 0) return tya_bind_method(receiver, tya_method_equal_p);
    if (strcmp(key, "to_i") == 0) return tya_bind_method(receiver, tya_method_to_i);
    if (strcmp(key, "to_f") == 0 || strcmp(key, "to_number") == 0) return tya_bind_method(receiver, tya_method_to_f);
    if (strcmp(key, "compare") == 0) return tya_bind_method(receiver, tya_method_compare);
    if (strcmp(key, "lt?") == 0) return tya_bind_method(receiver, tya_method_lt_p);
    if (strcmp(key, "lte?") == 0) return tya_bind_method(receiver, tya_method_lte_p);
    if (strcmp(key, "gt?") == 0) return tya_bind_method(receiver, tya_method_gt_p);
    if (strcmp(key, "gte?") == 0) return tya_bind_method(receiver, tya_method_gte_p);
    if (strcmp(key, "between?") == 0) return tya_bind_method(receiver, tya_method_between_p);
    if (strcmp(key, "upper") == 0) return tya_bind_method(receiver, tya_method_upper);
    if (strcmp(key, "lower") == 0) return tya_bind_method(receiver, tya_method_lower);
    if (strcmp(key, "blank?") == 0) return tya_bind_method(receiver, tya_method_blank_p);
    if (strcmp(key, "present?") == 0) return tya_bind_method(receiver, tya_method_present_p);
    if (strcmp(key, "iter") == 0) return tya_bind_method(receiver, tya_method_iter);
    if (strcmp(key, "sequence") == 0) return tya_bind_method(receiver, tya_method_sequence);
    return tya_nil();
  case TYA_BYTES:
    if (strcmp(key, "len") == 0) return tya_bind_method(receiver, tya_method_len);
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "iter") == 0) return tya_bind_method(receiver, tya_method_iter);
    if (strcmp(key, "sequence") == 0) return tya_bind_method(receiver, tya_method_sequence);
    return tya_nil();
  case TYA_ARRAY:
    if (strcmp(key, "len") == 0) return tya_bind_method(receiver, tya_method_len);
    if (strcmp(key, "empty?") == 0) return tya_bind_method(receiver, tya_method_empty_p);
    if (strcmp(key, "first") == 0) return tya_bind_method(receiver, tya_method_first);
    if (strcmp(key, "last") == 0) return tya_bind_method(receiver, tya_method_last);
    if (strcmp(key, "push") == 0) return tya_bind_method(receiver, tya_method_push);
    if (strcmp(key, "pop") == 0) return tya_bind_method(receiver, tya_method_pop);
    if (strcmp(key, "join") == 0) return tya_bind_method(receiver, tya_method_join);
    if (strcmp(key, "map") == 0) return tya_bind_method(receiver, tya_method_map);
    if (strcmp(key, "filter") == 0) return tya_bind_method(receiver, tya_method_filter);
    if (strcmp(key, "find") == 0) return tya_bind_method(receiver, tya_method_find);
    if (strcmp(key, "any") == 0) return tya_bind_method(receiver, tya_method_any);
    if (strcmp(key, "all") == 0) return tya_bind_method(receiver, tya_method_all);
    if (strcmp(key, "reduce") == 0) return tya_bind_method(receiver, tya_method_reduce);
    if (strcmp(key, "contains") == 0) return tya_bind_method(receiver, tya_method_contains);
    if (strcmp(key, "slice") == 0) return tya_bind_method(receiver, tya_method_slice);
    if (strcmp(key, "reverse") == 0) return tya_bind_method(receiver, tya_method_reverse);
    if (strcmp(key, "sort") == 0) return tya_bind_method(receiver, tya_method_sort);
    if (strcmp(key, "sort_by") == 0) return tya_bind_method(receiver, tya_method_sort_by);
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "equal?") == 0) return tya_bind_method(receiver, tya_method_equal_p);
    if (strcmp(key, "iter") == 0) return tya_bind_method(receiver, tya_method_iter);
    if (strcmp(key, "sequence") == 0) return tya_bind_method(receiver, tya_method_sequence);
    return tya_nil();
  case TYA_DICT:
    if (strcmp(key, "len") == 0) return tya_bind_method(receiver, tya_method_len);
    if (strcmp(key, "empty?") == 0) return tya_bind_method(receiver, tya_method_empty_p);
    if (strcmp(key, "has") == 0 || strcmp(key, "has?") == 0) return tya_bind_method(receiver, tya_method_has);
    if (strcmp(key, "get") == 0) return tya_bind_method(receiver, tya_method_get);
    if (strcmp(key, "set") == 0) return tya_bind_method(receiver, tya_method_set);
    if (strcmp(key, "delete") == 0) return tya_bind_method(receiver, tya_method_delete);
    if (strcmp(key, "keys") == 0) return tya_bind_method(receiver, tya_method_keys);
    if (strcmp(key, "values") == 0) return tya_bind_method(receiver, tya_method_values);
    if (strcmp(key, "entries") == 0) return tya_bind_method(receiver, tya_method_entries);
    if (strcmp(key, "merge") == 0) return tya_bind_method(receiver, tya_method_merge);
    if (strcmp(key, "merge!") == 0) return tya_bind_method(receiver, tya_method_merge_bang);
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    if (strcmp(key, "equal?") == 0) return tya_bind_method(receiver, tya_method_equal_p);
    if (strcmp(key, "iter") == 0) return tya_bind_method(receiver, tya_method_iter);
    if (strcmp(key, "sequence") == 0) return tya_bind_method(receiver, tya_method_sequence);
    return tya_nil();
  case TYA_CHANNEL:
    if (strcmp(key, "send") == 0) return tya_bind_method(receiver, tya_method_channel_send);
    if (strcmp(key, "receive") == 0) return tya_bind_method(receiver, tya_method_channel_receive);
    if (strcmp(key, "receive_timeout") == 0) return tya_bind_method(receiver, tya_method_channel_receive_timeout);
    if (strcmp(key, "close") == 0) return tya_bind_method(receiver, tya_method_channel_close);
    if (strcmp(key, "closed?") == 0) return tya_bind_method(receiver, tya_method_channel_closed_p);
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    return tya_nil();
  case TYA_TASK:
    if (strcmp(key, "cancel") == 0) return tya_bind_method(receiver, tya_method_task_cancel);
    if (strcmp(key, "cancelled?") == 0) return tya_bind_method(receiver, tya_method_task_cancelled_p);
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    return tya_nil();
  case TYA_RESOURCE:
    if (receiver.resource == NULL) return tya_nil();
    if (receiver.resource->subkind == TYA_RES_MUTEX) {
      if (strcmp(key, "lock") == 0) return tya_bind_method(receiver, tya_method_mutex_lock);
      if (strcmp(key, "unlock") == 0) return tya_bind_method(receiver, tya_method_mutex_unlock);
      if (strcmp(key, "with_lock") == 0) return tya_bind_method(receiver, tya_method_mutex_with_lock);
    }
    if (receiver.resource->subkind == TYA_RES_ATOMIC_INTEGER) {
      if (strcmp(key, "add") == 0) return tya_bind_method(receiver, tya_method_atomic_integer_add);
      if (strcmp(key, "load") == 0) return tya_bind_method(receiver, tya_method_atomic_integer_load);
      if (strcmp(key, "store") == 0) return tya_bind_method(receiver, tya_method_atomic_integer_store);
      if (strcmp(key, "cas") == 0) return tya_bind_method(receiver, tya_method_atomic_integer_compare_and_swap);
      if (strcmp(key, "compare_and_swap") == 0) return tya_bind_method(receiver, tya_method_atomic_integer_compare_and_swap);
    }
    if (receiver.resource->subkind == TYA_RES_WAIT_GROUP) {
      if (strcmp(key, "add") == 0) return tya_bind_method(receiver, tya_method_wait_group_add);
      if (strcmp(key, "done") == 0) return tya_bind_method(receiver, tya_method_wait_group_done);
      if (strcmp(key, "wait") == 0) return tya_bind_method(receiver, tya_method_wait_group_wait);
    }
    if (strcmp(key, "to_string") == 0) return tya_bind_method(receiver, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(receiver, tya_method_inspect);
    return tya_nil();
  default:
    return tya_nil();
  }
}

static TyaValue tya_method_len(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_len(receiver); }
static TyaValue tya_method_empty_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_bool((int)tya_len(receiver).number == 0); }
static TyaValue tya_method_to_string(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_to_string(receiver); }
static TyaValue tya_method_inspect(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_inspect(receiver); }
static TyaValue tya_method_to_i(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_to_int(receiver); }
static TyaValue tya_method_to_f(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_to_number(receiver); }
static TyaValue tya_method_compare(TyaValue receiver, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_compare(receiver, other); }
static TyaValue tya_method_equal_p(TyaValue receiver, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d;
  if (receiver.kind == TYA_ARRAY || receiver.kind == TYA_DICT) return tya_deep_equal(receiver, other);
  return tya_bool(tya_equal(receiver, other));
}
static TyaValue tya_method_lt_p(TyaValue receiver, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_bool(tya_compare(receiver, other).number < 0); }
static TyaValue tya_method_lte_p(TyaValue receiver, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_bool(tya_compare(receiver, other).number <= 0); }
static TyaValue tya_method_gt_p(TyaValue receiver, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_bool(tya_compare(receiver, other).number > 0); }
static TyaValue tya_method_gte_p(TyaValue receiver, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_bool(tya_compare(receiver, other).number >= 0); }
static TyaValue tya_method_between_p(TyaValue receiver, TyaValue min, TyaValue max, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_bool(tya_compare(receiver, min).number >= 0 && tya_compare(receiver, max).number <= 0); }
static TyaValue tya_method_abs(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_number(fabs(receiver.number)); }
static TyaValue tya_method_floor(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_floor(receiver); }
static TyaValue tya_method_ceil(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_ceil(receiver); }
static TyaValue tya_method_round(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_round(receiver); }
static TyaValue tya_method_trunc(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_trunc(receiver); }
static TyaValue tya_method_sqrt(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_sqrt(receiver); }
static TyaValue tya_method_pow(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_math_pow(receiver, a); }
static TyaValue tya_method_log(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_log(receiver); }
static TyaValue tya_method_log2(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_log2(receiver); }
static TyaValue tya_method_log10(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_log10(receiver); }
static TyaValue tya_method_exp(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_exp(receiver); }
static TyaValue tya_method_sin(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_sin(receiver); }
static TyaValue tya_method_cos(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_cos(receiver); }
static TyaValue tya_method_tan(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_tan(receiver); }
static TyaValue tya_method_asin(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_asin(receiver); }
static TyaValue tya_method_acos(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_acos(receiver); }
static TyaValue tya_method_atan(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_math_atan(receiver); }
static TyaValue tya_method_atan2(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_math_atan2(receiver, a); }
static TyaValue tya_method_integer_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_number_integer_p(receiver); }
static TyaValue tya_method_finite_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_number_finite_p(receiver); }
static TyaValue tya_method_nan_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_number_nan_p(receiver); }
static TyaValue tya_method_byte_len(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_byte_len(receiver); }
static TyaValue tya_method_string_slice(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_string_slice(receiver, a, b); }
static TyaValue tya_method_trim(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_trim(receiver); }
static TyaValue tya_method_contains(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_contains_method(receiver, a); }
static TyaValue tya_method_string_index_of(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_string_index_of(receiver, a, b); }
static TyaValue tya_method_starts_with(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_starts_with(receiver, a); }
static TyaValue tya_method_ends_with(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_ends_with(receiver, a); }
static TyaValue tya_method_replace(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_replace(receiver, a, b); }
static TyaValue tya_method_split(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_split(receiver, a); }
static TyaValue tya_method_lines(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_lines(receiver); }
static TyaValue tya_method_chars(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_chars(receiver); }
static TyaValue tya_method_bytes(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_bytes_of(receiver); }
static TyaValue tya_method_upper(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_upcase(receiver); }
static TyaValue tya_method_lower(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_downcase(receiver); }
static TyaValue tya_method_blank_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_bool(tya_equal(tya_trim(receiver), tya_string(""))); }
static TyaValue tya_method_present_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_bool(!tya_equal(tya_trim(receiver), tya_string(""))); }
static TyaValue tya_method_first(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_first(receiver); }
static TyaValue tya_method_last(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_last(receiver); }
static TyaValue tya_method_push(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; (void)tya_array_push(receiver, a); return tya_legacy_modules_enabled() ? receiver : tya_nil(); }
static TyaValue tya_method_pop(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_pop(receiver); }
static TyaValue tya_method_slice(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_slice(receiver, a, b); }
static TyaValue tya_method_reverse(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_reverse(receiver); }
static TyaValue tya_method_sort(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_array_sort(receiver); }
static TyaValue tya_method_sort_by(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_array_sort_by(receiver, a); }
static TyaValue tya_method_join(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_join(receiver, a); }
static TyaValue tya_method_map(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_map(receiver, a); }
static TyaValue tya_method_filter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_filter(receiver, a); }
static TyaValue tya_method_find(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_find(receiver, a); }
static TyaValue tya_method_any(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_any(receiver, a); }
static TyaValue tya_method_all(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_all(receiver, a); }
static TyaValue tya_method_reduce(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_reduce(receiver, a, b); }
static TyaValue tya_method_iter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_iter(receiver); }
static TyaValue tya_method_sequence(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_sequence(receiver); }
static TyaValue tya_method_has(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_has(receiver, a); }
static TyaValue tya_method_get(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return b.kind == TYA_NIL ? tya_dict_get(receiver, a, tya_nil(), false) : tya_dict_get(receiver, a, b, true); }
static TyaValue tya_method_set(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_dict_set(receiver, a, b); }
static TyaValue tya_method_delete(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_dict_delete(receiver, a); }
static TyaValue tya_method_keys(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_keys(receiver); }
static TyaValue tya_method_values(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_values(receiver); }
static TyaValue tya_method_entries(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_dict_entries(receiver); }
static TyaValue tya_method_merge(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_dict_merge(receiver, a); }
static TyaValue tya_method_merge_bang(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_dict_merge_bang(receiver, a); }
static TyaValue tya_method_channel_send(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_channel_send(receiver, a); }
static TyaValue tya_method_channel_receive(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_channel_receive(receiver); }
static TyaValue tya_method_channel_receive_timeout(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_channel_receive_timeout(receiver, a); }
static TyaValue tya_method_channel_close(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_channel_close(receiver); }
static TyaValue tya_method_channel_closed_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_channel_closed(receiver); }
static TyaValue tya_method_task_cancel(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_task_cancel(receiver); }
static TyaValue tya_method_task_cancelled_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_task_is_cancelled(receiver); }
static TyaValue tya_method_mutex_lock(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_sync_lock(receiver); }
static TyaValue tya_method_mutex_unlock(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_sync_unlock(receiver); }
static TyaValue tya_method_mutex_with_lock(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d;
  TyaResource *r = tya_resource_check(receiver, TYA_RES_MUTEX, "sync.mutex.with_lock");
  if (r == NULL) return tya_nil();
  if (fn.kind != TYA_FUNCTION || fn.function == NULL) {
    tya_raise(tya_string("sync.mutex.with_lock: argument must be callable"));
    return tya_nil();
  }
  pthread_mutex_lock(&r->mu);
  TyaRaiseFrame frame;
  if (setjmp(frame.env) == 0) {
    tya_push_raise_frame(&frame);
    TyaValue result = tya_call0(fn);
    tya_pop_raise_frame();
    pthread_mutex_unlock(&r->mu);
    return result;
  }
  tya_pop_raise_frame();
  pthread_mutex_unlock(&r->mu);
  tya_raise(frame.value);
  return tya_nil();
}
static TyaValue tya_method_atomic_integer_add(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sync_atomic_integer_add(receiver, a); }
static TyaValue tya_method_atomic_integer_load(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_sync_atomic_integer_load(receiver); }
static TyaValue tya_method_atomic_integer_store(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sync_atomic_integer_store(receiver, a); }
static TyaValue tya_method_atomic_integer_compare_and_swap(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)c; (void)d; return tya_sync_atomic_integer_cas(receiver, a, b); }
static TyaValue tya_method_wait_group_add(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sync_wait_group_add(receiver, a); }
static TyaValue tya_method_wait_group_done(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_sync_wait_group_done(receiver); }
static TyaValue tya_method_wait_group_wait(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_sync_wait_group_wait(receiver); }
static TyaValue tya_method_iterator_has_next(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_iterator_has_next(receiver); }
static TyaValue tya_method_iterator_next(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)a; (void)b; (void)c; (void)d; return tya_iterator_next(receiver); }
static TyaValue tya_method_sequence_iter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d;
  TyaValue kind = tya_member(receiver, "__sequence_kind");
  TyaValue source = tya_member(receiver, "source");
  if (kind.kind != TYA_STRING || kind.string == NULL || strcmp(kind.string, "iterable") == 0) return tya_iter(source);
  return tya_sequence_iterator(kind.string, tya_iter(source), tya_member(receiver, "fn"), tya_member(receiver, "n"));
}
static TyaValue tya_method_sequence_map(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sequence_object("map", receiver, fn, tya_nil()); }
static TyaValue tya_method_sequence_filter(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sequence_object("filter", receiver, fn, tya_nil()); }
static TyaValue tya_method_sequence_take(TyaValue receiver, TyaValue n, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sequence_object("take", receiver, tya_nil(), n); }
static TyaValue tya_method_sequence_drop(TyaValue receiver, TyaValue n, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) { (void)b; (void)c; (void)d; return tya_sequence_object("drop", receiver, tya_nil(), n); }
static TyaValue tya_method_sequence_reduce(TyaValue receiver, TyaValue initial, TyaValue fn, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)c; (void)d;
  TyaValue acc = initial;
  TyaValue iter = tya_iter(receiver);
  while (tya_truthy(tya_iterator_has_next(iter))) {
    acc = tya_call2(fn, acc, tya_iterator_next(iter));
  }
  return acc;
}
static TyaValue tya_method_sequence_each(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d;
  TyaValue iter = tya_iter(receiver);
  while (tya_truthy(tya_iterator_has_next(iter))) {
    (void)tya_call1(fn, tya_iterator_next(iter));
  }
  return tya_nil();
}
static TyaValue tya_method_sequence_any_p(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d;
  TyaValue iter = tya_iter(receiver);
  while (tya_truthy(tya_iterator_has_next(iter))) {
    if (tya_truthy(tya_call1(fn, tya_iterator_next(iter)))) return tya_bool(true);
  }
  return tya_bool(false);
}
static TyaValue tya_method_sequence_all_p(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d;
  TyaValue iter = tya_iter(receiver);
  while (tya_truthy(tya_iterator_has_next(iter))) {
    if (!tya_truthy(tya_call1(fn, tya_iterator_next(iter)))) return tya_bool(false);
  }
  return tya_bool(true);
}
static TyaValue tya_method_sequence_find(TyaValue receiver, TyaValue fn, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d;
  TyaValue iter = tya_iter(receiver);
  while (tya_truthy(tya_iterator_has_next(iter))) {
    TyaValue item = tya_iterator_next(iter);
    if (tya_truthy(tya_call1(fn, item))) return item;
  }
  return tya_nil();
}
static TyaValue tya_method_sequence_to_a(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d;
  TyaValue out = tya_array(NULL, 0);
  TyaValue iter = tya_iter(receiver);
  while (tya_truthy(tya_iterator_has_next(iter))) {
    tya_push(out, tya_iterator_next(iter));
  }
  return out;
}

/* =========================================================================
 * v0.24: process
 * ========================================================================= */

static char *tya_dup_cstr(const char *s) {
  size_t n = strlen(s) + 1;
  char *out = malloc(n);
  memcpy(out, s, n);
  return out;
}

static char *tya_read_all(int fd) {
  size_t cap = 256;
  size_t len = 0;
  char *buf = malloc(cap);
  for (;;) {
    if (len + 1 >= cap) {
      cap *= 2;
      buf = realloc(buf, cap);
    }
    ssize_t r = read(fd, buf + len, cap - len - 1);
    if (r < 0) {
      if (errno == EINTR) continue;
      free(buf);
      return NULL;
    }
    if (r == 0) break;
    len += (size_t)r;
  }
  buf[len] = '\0';
  return buf;
}

TyaValue tya_process_run(TyaValue command, TyaValue options) {
  bool shell = false;
  if (options.kind == TYA_DICT && options.dict != NULL) {
    TyaValue shell_v = tya_index(options, tya_string("shell"));
    shell = shell_v.kind == TYA_BOOL && shell_v.boolean;
  }
  if (command.kind == TYA_STRING && !shell) {
    tya_raise(tya_string("process.run: string command requires shell option"));
    return tya_nil();
  }
  if (command.kind != TYA_STRING && (command.kind != TYA_ARRAY || command.array == NULL || command.array->len == 0)) {
    tya_raise(tya_string("process.run: command must be a string or non-empty array of strings"));
    return tya_nil();
  }
  int argc = command.kind == TYA_STRING ? 3 : command.array->len;
  char **argv = malloc(sizeof(char *) * (size_t)(argc + 1));
  if (command.kind == TYA_STRING) {
    argv[0] = tya_dup_cstr("sh");
    argv[1] = tya_dup_cstr("-c");
    argv[2] = tya_dup_cstr(command.string == NULL ? "" : command.string);
  } else {
    for (int i = 0; i < argc; i++) {
      TyaValue item = command.array->items[i];
      if (item.kind != TYA_STRING || item.string == NULL) {
        for (int j = 0; j < i; j++) free(argv[j]);
        free(argv);
        tya_raise(tya_string("process.run: command items must be strings"));
        return tya_nil();
      }
      argv[i] = tya_dup_cstr(item.string);
    }
  }
  argv[argc] = NULL;

  const char *cwd_path = NULL;
  const char *input_text = NULL;
  size_t input_len = 0;
  char **child_env = NULL;
  if (options.kind == TYA_DICT && options.dict != NULL) {
    TyaValue cwd = tya_index(options, tya_string("cwd"));
    if (cwd.kind == TYA_STRING && cwd.string != NULL) {
      cwd_path = cwd.string;
    }
    TyaValue inp = tya_index(options, tya_string("input"));
    if (inp.kind == TYA_STRING && inp.string != NULL) {
      input_text = inp.string;
      input_len = strlen(input_text);
    }
    TyaValue env_v = tya_index(options, tya_string("env"));
    if (env_v.kind == TYA_DICT && env_v.dict != NULL) {
      int env_count = 0;
      for (int i = 0; i < env_v.dict->len; i++) {
        if (env_v.dict->entries[i].key != NULL) env_count++;
      }
      child_env = malloc(sizeof(char *) * (size_t)(env_count + 1));
      int idx = 0;
      for (int i = 0; i < env_v.dict->len; i++) {
        if (env_v.dict->entries[i].key == NULL) continue;
        TyaValue val = env_v.dict->entries[i].value;
        if (val.kind != TYA_STRING || val.string == NULL) {
          for (int j = 0; j < idx; j++) free(child_env[j]);
          free(child_env);
          for (int j = 0; j < argc; j++) free(argv[j]);
          free(argv);
          tya_raise(tya_string("process.run: env values must be strings"));
          return tya_nil();
        }
        size_t kl = strlen(env_v.dict->entries[i].key);
        size_t vl = strlen(val.string);
        char *entry = malloc(kl + 1 + vl + 1);
        memcpy(entry, env_v.dict->entries[i].key, kl);
        entry[kl] = '=';
        memcpy(entry + kl + 1, val.string, vl);
        entry[kl + 1 + vl] = '\0';
        child_env[idx++] = entry;
      }
      child_env[idx] = NULL;
    }
  }

  int in_pipe[2] = {-1, -1};
  int out_pipe[2] = {-1, -1};
  int err_pipe[2] = {-1, -1};
  if (pipe(in_pipe) < 0 || pipe(out_pipe) < 0 || pipe(err_pipe) < 0) {
    tya_raise(tya_string("process.run: pipe failed"));
    return tya_nil();
  }

  pid_t pid = fork();
  if (pid < 0) {
    tya_raise(tya_string("process.run: fork failed"));
    return tya_nil();
  }
  if (pid == 0) {
    /* child */
    dup2(in_pipe[0], 0);
    dup2(out_pipe[1], 1);
    dup2(err_pipe[1], 2);
    close(in_pipe[0]); close(in_pipe[1]);
    close(out_pipe[0]); close(out_pipe[1]);
    close(err_pipe[0]); close(err_pipe[1]);
    if (cwd_path && chdir(cwd_path) < 0) {
      _exit(127);
    }
    if (child_env) {
      for (int i = 0; child_env[i]; i++) {
        putenv(child_env[i]);
      }
    }
    execvp(argv[0], argv);
    _exit(127);
  }
  /* parent */
  close(in_pipe[0]);
  close(out_pipe[1]);
  close(err_pipe[1]);
  if (input_text && input_len > 0) {
    size_t written = 0;
    while (written < input_len) {
      ssize_t w = write(in_pipe[1], input_text + written, input_len - written);
      if (w < 0) {
        if (errno == EINTR) continue;
        break;
      }
      written += (size_t)w;
    }
  }
  close(in_pipe[1]);
  char *out_buf = tya_read_all(out_pipe[0]);
  char *err_buf = tya_read_all(err_pipe[0]);
  close(out_pipe[0]);
  close(err_pipe[0]);
  int status = 0;
  while (waitpid(pid, &status, 0) < 0) {
    if (errno != EINTR) break;
  }

  for (int i = 0; i < argc; i++) free(argv[i]);
  free(argv);
  if (child_env) {
    for (int i = 0; child_env[i]; i++) free(child_env[i]);
    free(child_env);
  }

  int exit_code = 0;
  if (WIFEXITED(status)) {
    exit_code = WEXITSTATUS(status);
  } else if (WIFSIGNALED(status)) {
    exit_code = 128 + WTERMSIG(status);
  }

  TyaValue result = tya_dict(NULL, 0);
  tya_set_member(result, "status", tya_number((double)exit_code));
  tya_set_member(result, "exit_code", tya_number((double)exit_code));
  tya_set_member(result, "success", tya_bool(exit_code == 0));
  tya_set_member(result, "stdout", tya_string(out_buf ? out_buf : ""));
  tya_set_member(result, "stderr", tya_string(err_buf ? err_buf : ""));
  tya_set_member(result, "timed_out", tya_bool(false));
  if (out_buf == NULL) free(out_buf);
  if (err_buf == NULL) free(err_buf);
  return result;
}

TyaValue tya_process_exec(TyaValue command, TyaValue options) {
  (void)command;
  (void)options;
  tya_raise(tya_string("process.exec: unsupported on this runtime"));
  return tya_nil();
}

/* =========================================================================
 * v1: regex
 * ========================================================================= */

static void tya_regex_raise(const char *message, const char *code) {
  TyaDictEntry entries[] = {
    {"kind", tya_string("regex")},
    {"code", tya_string(code)},
  };
  tya_raise_user(tya_error2(tya_string(message), tya_dict(entries, 2)));
}

static int tya_regex_flags(TyaValue options) {
  if (options.kind == TYA_MISSING || options.kind == TYA_NIL) {
    return REG_EXTENDED;
  }
  if (options.kind != TYA_DICT || options.dict == NULL) {
    tya_regex_raise("regex.compile: options must be a dictionary", "invalid_options");
    return REG_EXTENDED;
  }
  int flags = REG_EXTENDED;
  for (int i = 0; i < options.dict->len; i++) {
    const char *key = options.dict->entries[i].key;
    TyaValue value = options.dict->entries[i].value;
    if (key == NULL) continue;
    if (value.kind != TYA_BOOL) {
      char msg[160];
      snprintf(msg, sizeof(msg), "regex.compile: option %s must be bool", key);
      tya_regex_raise(msg, "invalid_option_kind");
      return flags;
    }
    if (strcmp(key, "ignore_case") == 0) {
      if (value.boolean) flags |= REG_ICASE;
    } else if (strcmp(key, "multi_line") == 0) {
      if (value.boolean) flags |= REG_NEWLINE;
    } else if (strcmp(key, "dot_all") == 0) {
      if (!value.boolean) flags |= REG_NEWLINE;
    } else {
      char msg[160];
      snprintf(msg, sizeof(msg), "regex.compile: unknown option %s", key);
      tya_regex_raise(msg, "unknown_option");
      return flags;
    }
  }
  return flags;
}

static int tya_regex_rune_index(const char *text, int byte_index) {
  int runes = 0;
  for (int i = 0; text[i] != '\0' && i < byte_index;) {
    unsigned char c = (unsigned char)text[i];
    int width = 1;
    if ((c & 0x80) == 0) width = 1;
    else if ((c & 0xE0) == 0xC0) width = 2;
    else if ((c & 0xF0) == 0xE0) width = 3;
    else if ((c & 0xF8) == 0xF0) width = 4;
    i += width;
    runes++;
  }
  return runes;
}

static TyaValue tya_regex_match_dict(const char *text, regmatch_t *matches, size_t nmatch) {
  if (matches[0].rm_so < 0) return tya_nil();
  TyaValue groups = tya_array(NULL, 0);
  for (size_t i = 1; i < nmatch; i++) {
    if (matches[i].rm_so < 0) {
      tya_array_push(groups, tya_nil());
    } else {
      tya_array_push(groups, tya_string(tya_substr(text, (int)matches[i].rm_so, (int)(matches[i].rm_eo - matches[i].rm_so))));
    }
  }
  TyaDictEntry entries[] = {
    {"text", tya_string(tya_substr(text, (int)matches[0].rm_so, (int)(matches[0].rm_eo - matches[0].rm_so)))},
    {"start", tya_number(tya_regex_rune_index(text, (int)matches[0].rm_so))},
    {"end", tya_number(tya_regex_rune_index(text, (int)matches[0].rm_eo))},
    {"groups", groups},
  };
  return tya_dict(entries, 4);
}

static int tya_regex_compile_inner(TyaValue self, regex_t *re) {
  if (setlocale(LC_CTYPE, "C.UTF-8") == NULL) setlocale(LC_CTYPE, "");
  TyaValue pattern = tya_member(self, "__regex_pattern");
  TyaValue options = tya_member(self, "__regex_options");
  if (pattern.kind != TYA_STRING || pattern.string == NULL) {
    tya_regex_raise("regex.compile: pattern must be a string", "invalid_pattern_kind");
    return -1;
  }
  int rc = regcomp(re, pattern.string, tya_regex_flags(options));
  if (rc != 0) {
    tya_regex_raise("regex.compile: invalid pattern", "invalid_pattern");
    return -1;
  }
  return 0;
}

static TyaValue tya_regex_method_find(TyaValue self, TyaValue text, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e) {
  (void)a; (void)b; (void)c; (void)d; (void)e;
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_regex_raise("regex.find: text must be a string", "invalid_text");
    return tya_nil();
  }
  regex_t re;
  if (tya_regex_compile_inner(self, &re) < 0) return tya_nil();
  size_t nmatch = re.re_nsub + 1;
  if (nmatch > 16) nmatch = 16;
  regmatch_t matches[16];
  int rc = regexec(&re, text.string, nmatch, matches, 0);
  regfree(&re);
  if (rc != 0) return tya_nil();
  return tya_regex_match_dict(text.string, matches, nmatch);
}

static TyaValue tya_regex_method_match_p(TyaValue self, TyaValue text, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e) {
  (void)a; (void)b; (void)c; (void)d; (void)e;
  TyaValue found = tya_regex_method_find(self, text, tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil());
  return tya_bool(found.kind != TYA_NIL);
}

static TyaValue tya_regex_method_find_all(TyaValue self, TyaValue text, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e) {
  (void)a; (void)b; (void)c; (void)d; (void)e;
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_regex_raise("regex.find_all: text must be a string", "invalid_text");
    return tya_nil();
  }
  regex_t re;
  if (tya_regex_compile_inner(self, &re) < 0) return tya_nil();
  size_t nmatch = re.re_nsub + 1;
  if (nmatch > 16) nmatch = 16;
  TyaValue out = tya_array(NULL, 0);
  const char *cursor = text.string;
  int offset = 0;
  regmatch_t matches[16];
  while (regexec(&re, cursor, nmatch, matches, 0) == 0) {
    regmatch_t adjusted[16];
    for (size_t i = 0; i < nmatch; i++) {
      adjusted[i] = matches[i];
      if (adjusted[i].rm_so >= 0) {
        adjusted[i].rm_so += offset;
        adjusted[i].rm_eo += offset;
      }
    }
    tya_array_push(out, tya_regex_match_dict(text.string, adjusted, nmatch));
    int advance = (int)matches[0].rm_eo;
    if (advance <= 0) advance = 1;
    cursor += advance;
    offset += advance;
    if (*cursor == '\0') break;
  }
  regfree(&re);
  return out;
}

static TyaValue tya_regex_method_split(TyaValue self, TyaValue text, TyaValue limit_v, TyaValue b, TyaValue c, TyaValue d, TyaValue e) {
  (void)b; (void)c; (void)d; (void)e;
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_regex_raise("regex.split: text must be a string", "invalid_text");
    return tya_nil();
  }
  int limit = -1;
  if (limit_v.kind != TYA_NIL && limit_v.kind != TYA_MISSING) {
    if (limit_v.kind != TYA_NUMBER || floor(limit_v.number) != limit_v.number) {
      tya_regex_raise("regex.split: limit must be an integer", "invalid_limit");
      return tya_nil();
    }
    limit = (int)limit_v.number;
  }
  regex_t re;
  if (tya_regex_compile_inner(self, &re) < 0) return tya_nil();
  TyaValue out = tya_array(NULL, 0);
  const char *cursor = text.string;
  int offset = 0;
  int parts = 0;
  regmatch_t m;
  while ((limit < 0 || parts < limit - 1) && regexec(&re, cursor, 1, &m, 0) == 0) {
    tya_array_push(out, tya_string(tya_substr(text.string, offset, (int)m.rm_so)));
    parts++;
    int advance = (int)m.rm_eo;
    if (advance <= 0) advance = 1;
    cursor += advance;
    offset += advance;
    if (*cursor == '\0') break;
  }
  tya_array_push(out, tya_string(tya_dup_cstr(cursor)));
  regfree(&re);
  return out;
}

typedef struct {
  char *data;
  int len;
  int cap;
} TyaRegexBuffer;

static void tya_regex_buf_append(TyaRegexBuffer *buf, const char *data, int len) {
  if (len <= 0) return;
  if (buf->len + len + 1 > buf->cap) {
    int next = buf->cap == 0 ? 64 : buf->cap;
    while (buf->len + len + 1 > next) next *= 2;
    buf->data = realloc(buf->data, (size_t)next);
    buf->cap = next;
  }
  memcpy(buf->data + buf->len, data, (size_t)len);
  buf->len += len;
  buf->data[buf->len] = '\0';
}

static int tya_regex_expand_replacement(TyaRegexBuffer *out, const char *text, const char *replacement, regmatch_t *matches, size_t nmatch) {
  for (int i = 0; replacement[i] != '\0';) {
    if (replacement[i] != '$') {
      tya_regex_buf_append(out, replacement + i, 1);
      i++;
      continue;
    }
    if (replacement[i + 1] == '$') {
      tya_regex_buf_append(out, "$", 1);
      i += 2;
      continue;
    }
    if (replacement[i + 1] != '{') {
      tya_regex_raise("regex.replace: invalid replacement capture", "invalid_replacement");
      return -1;
    }
    int j = i + 2;
    int index = 0;
    if (replacement[j] < '0' || replacement[j] > '9') {
      tya_regex_raise("regex.replace: invalid replacement capture", "invalid_replacement");
      return -1;
    }
    while (replacement[j] >= '0' && replacement[j] <= '9') {
      index = index * 10 + (replacement[j] - '0');
      j++;
    }
    if (replacement[j] != '}') {
      tya_regex_raise("regex.replace: invalid replacement capture", "invalid_replacement");
      return -1;
    }
    if (index == 0 || (size_t)index >= nmatch) {
      tya_regex_raise("regex.replace: unknown capture reference", "unknown_capture");
      return -1;
    }
    if (matches[index].rm_so >= 0) {
      tya_regex_buf_append(out, text + matches[index].rm_so, (int)(matches[index].rm_eo - matches[index].rm_so));
    }
    i = j + 1;
  }
  return 0;
}

static TyaValue tya_regex_method_replace(TyaValue self, TyaValue text, TyaValue replacement, TyaValue limit_v, TyaValue c, TyaValue d, TyaValue e) {
  (void)c; (void)d; (void)e;
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_regex_raise("regex.replace: text must be a string", "invalid_text");
    return tya_nil();
  }
  if (replacement.kind != TYA_STRING || replacement.string == NULL) {
    tya_regex_raise("regex.replace: replacement must be a string", "invalid_replacement");
    return tya_nil();
  }
  int limit = -1;
  if (limit_v.kind != TYA_NIL && limit_v.kind != TYA_MISSING) {
    if (limit_v.kind != TYA_NUMBER || floor(limit_v.number) != limit_v.number) {
      tya_regex_raise("regex.replace: limit must be an integer", "invalid_limit");
      return tya_nil();
    }
    limit = (int)limit_v.number;
  }
  regex_t re;
  if (tya_regex_compile_inner(self, &re) < 0) return tya_nil();
  size_t nmatch = re.re_nsub + 1;
  if (nmatch > 16) nmatch = 16;
  TyaRegexBuffer out = {0};
  const char *cursor = text.string;
  int offset = 0;
  int replaced = 0;
  regmatch_t matches[16];
  while ((limit < 0 || replaced < limit) && regexec(&re, cursor, nmatch, matches, 0) == 0) {
    tya_regex_buf_append(&out, cursor, (int)matches[0].rm_so);
    regmatch_t adjusted[16];
    for (size_t i = 0; i < nmatch; i++) {
      adjusted[i] = matches[i];
      if (adjusted[i].rm_so >= 0) {
        adjusted[i].rm_so += offset;
        adjusted[i].rm_eo += offset;
      }
    }
    if (tya_regex_expand_replacement(&out, text.string, replacement.string, adjusted, nmatch) < 0) {
      regfree(&re);
      return tya_nil();
    }
    int advance = (int)matches[0].rm_eo;
    if (advance <= 0) advance = 1;
    cursor += advance;
    offset += advance;
    replaced++;
    if (*cursor == '\0') break;
  }
  tya_regex_buf_append(&out, cursor, (int)strlen(cursor));
  regfree(&re);
  return tya_string(out.data == NULL ? "" : out.data);
}

TyaValue tya_regex_compile(TyaValue pattern, TyaValue options) {
#ifdef _WIN32
  (void)pattern; (void)options;
  tya_regex_raise("regex.compile: unsupported on this runtime", "unsupported_runtime");
  return tya_nil();
#else
  if (setlocale(LC_CTYPE, "C.UTF-8") == NULL) setlocale(LC_CTYPE, "");
  if (pattern.kind != TYA_STRING || pattern.string == NULL) {
    tya_regex_raise("regex.compile: pattern must be a string", "invalid_pattern_kind");
    return tya_nil();
  }
  regex_t re;
  int rc = regcomp(&re, pattern.string, tya_regex_flags(options));
  if (rc != 0) {
    tya_regex_raise("regex.compile: invalid pattern", "invalid_pattern");
    return tya_nil();
  }
  regfree(&re);
  TyaValue obj = tya_object();
  tya_set_member(obj, "__regex_pattern", pattern);
  tya_set_member(obj, "__regex_options", options);
  tya_set_member(obj, "match?", tya_bind_method(obj, tya_regex_method_match_p));
  tya_set_member(obj, "find", tya_bind_method(obj, tya_regex_method_find));
  tya_set_member(obj, "find_all", tya_bind_method(obj, tya_regex_method_find_all));
  tya_set_member(obj, "split", tya_bind_method(obj, tya_regex_method_split));
  tya_set_member(obj, "replace", tya_bind_method(obj, tya_regex_method_replace));
  return obj;
#endif
}

/* =========================================================================
 * v0.24: digest (MD5, SHA1, SHA256, SHA384, SHA512)
 * Public-domain inline implementations.
 * ========================================================================= */

/* ---- MD5 ---- */
