if exists("b:current_syntax")
  finish
endif

syntax case match

syntax match tyaComment "#.*$"

syntax region tyaString start=/"/ skip=/\\./ end=/"/ contains=tyaInterpolation
syntax region tyaTripleString start=/"""/ end=/"""/ contains=tyaInterpolation
syntax region tyaBytes start=/b"/ skip=/\\./ end=/"/ contains=tyaInterpolation
syntax region tyaInterpolation start=/{/ end=/}/ contained contains=tyaNumber,tyaKeyword,tyaLiteral,tyaOperator

syntax match tyaNumber "\v<0x[0-9a-fA-F_]+>"
syntax match tyaNumber "\v<0b[01_]+>"
syntax match tyaNumber "\v<[0-9][0-9_]*(\.[0-9][0-9_]*)?>"

syntax keyword tyaKeyword if elseif else while for in break continue return raise try catch match case when select receive send timeout default
syntax keyword tyaDeclaration class module interface implements extends abstract final private static initialize import as
syntax keyword tyaConcurrency spawn await scope
syntax keyword tyaLiteral true false nil
syntax keyword tyaSelf self Self super
syntax keyword tyaLogical and or not

syntax match tyaTypeDecl "\<\(class\|interface\)\s\+\zs[A-Z][A-Za-z0-9_]*"
syntax match tyaModuleDecl "\<module\s\+\zs[A-Za-z_][A-Za-z0-9_]*"
syntax match tyaFunctionDecl "\<[A-Za-z_][A-Za-z0-9_?]*\>\ze\s*=\s*->"
syntax match tyaOperator "->\|==\|!=\|<=\|>=\|<<\|>>\|[+\-*\/%=<>.,:&|^~]"

highlight default link tyaComment Comment
highlight default link tyaString String
highlight default link tyaTripleString String
highlight default link tyaBytes String
highlight default link tyaInterpolation Special
highlight default link tyaNumber Number
highlight default link tyaKeyword Keyword
highlight default link tyaDeclaration Structure
highlight default link tyaConcurrency Keyword
highlight default link tyaLiteral Constant
highlight default link tyaSelf Identifier
highlight default link tyaLogical Operator
highlight default link tyaTypeDecl Type
highlight default link tyaModuleDecl Identifier
highlight default link tyaFunctionDecl Function
highlight default link tyaOperator Operator

let b:current_syntax = "tya"
