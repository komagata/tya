if exists("b:did_indent")
  finish
endif
let b:did_indent = 1

setlocal autoindent
setlocal expandtab
setlocal shiftwidth=2
setlocal softtabstop=2

let b:undo_indent = "setlocal autoindent< expandtab< shiftwidth< softtabstop<"
