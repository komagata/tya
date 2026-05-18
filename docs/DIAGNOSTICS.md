# Tya Diagnostic Reference

This reference lists the public `TYA-E....` diagnostic codes currently emitted
by the compiler, runtime, CLI, package tooling, and LSP-facing paths. Each code
has a short explanation so release checks can verify that stable diagnostics
are discoverable without requiring tutorial-length pages for every code.

| Code | Summary |
| --- | --- |
| TYA-E0001 | Tab indentation is invalid. |
| TYA-E0002 | Indentation increased where no block was opened. |
| TYA-E0003 | Indentation decreased to a non-existent block level. |
| TYA-E0004 | Indentation is inconsistent. |
| TYA-E0005 | Indentation step is invalid. |
| TYA-E0006 | String literal is unterminated. |
| TYA-E0007 | String escape is unterminated. |
| TYA-E0008 | String escape is unknown. |
| TYA-E0015 | Source contains an unexpected character. |
| TYA-E0016 | Heredoc string marker is invalid. |
| TYA-E0017 | Heredoc string indentation is mixed or too shallow. |
| TYA-E0018 | Heredoc string is unterminated. |
| TYA-E0019 | Raw string is unterminated. |
| TYA-E0020 | String interpolation is unterminated. |
| TYA-E0021 | String interpolation is invalid. |
| TYA-E0100 | Parser expected a token or syntax form. |
| TYA-E0101 | Parser expected an indented block. |
| TYA-E0102 | Parser found an unexpected token. |
| TYA-E0120 | Statement appears in an invalid position. |
| TYA-E0140 | Reserved name or keyword used as an identifier. |
| TYA-E0141 | Unsupported or removed syntax was used. |
| TYA-E0150 | Documentation comment is not attached to a declaration. |
| TYA-E0160 | Pattern or multi-assignment syntax is invalid. |
| TYA-E0180 | Parser expected an expression. |
| TYA-E0200 | Removed `module` declaration syntax was used. |
| TYA-E0301 | Strict checker rejected an invalid operation. |
| TYA-E0302 | Strict checker rejected invalid indexing. |
| TYA-E0303 | Strict checker rejected invalid arithmetic or comparison. |
| TYA-E0305 | Strict checker rejected invalid call or member usage. |
| TYA-E0306 | Strict checker rejected invalid assignment target. |
| TYA-E0307 | Strict checker rejected kind-changing reassignment. |
| TYA-E0308 | Strict checker rejected constant reassignment or mutation. |
| TYA-E0400 | Class file is missing its required public class. |
| TYA-E0402 | Class file contains unsupported top-level declarations. |
| TYA-E0403 | Class file imports are not before class/interface declarations. |
| TYA-E0404 | Class file name is not PascalCase. |
| TYA-E0405 | Class file declares the same public class more than once. |
| TYA-E0406 | Private class visibility was violated. |
| TYA-E0407 | Underscore privacy marker was used. |
| TYA-E0410 | Legacy instance variable syntax was used. |
| TYA-E0411 | `self` was used where no instance receiver exists. |
| TYA-E0412 | `Self` was used outside a class body. |
| TYA-E0413 | Class member access is not canonical. |
| TYA-E0414 | Removed constructor name syntax was used. |
| TYA-E0601 | C emitter rejected an unsupported assignment target. |
| TYA-E0602 | C emitter rejected an unsupported statement or expression. |
| TYA-E0603 | C emitter rejected an unsupported match pattern. |
| TYA-E0604 | C emitter rejected `try` outside a function body. |
| TYA-E0605 | C emitter rejected non-tuple multi-assignment. |
| TYA-E0606 | C emitter rejected a destructuring target. |
| TYA-E0610 | Embedded asset was not found. |
| TYA-E0611 | Embedded glob matched no files. |
| TYA-E0612 | Embedded asset transform is unknown. |
| TYA-E0613 | Embedded asset could not be read or transformed. |
| TYA-E0800 | Runner reported a general runtime/tooling failure. |
| TYA-E0810 | Removed `kind` builtin was used. |
| TYA-E0811 | Removed primitive helper module was imported. |
| TYA-E0812 | Removed top-level primitive helper was used. |
| TYA-E0813 | Code attempted to inherit from a primitive class. |
| TYA-E0814 | Code attempted to add or redefine a primitive class method. |
| TYA-E0815 | Code attempted to rebind a reserved primitive class identifier. |
| TYA-E0820 | Removed concurrency helper API was used. |
| TYA-E0840 | Entry filename is invalid for the requested operation. |
| TYA-E0850 | Class file was used where a script file is required. |
| TYA-E0851 | Module name is invalid. |
| TYA-E0852 | Package contains a script file. |
| TYA-E0853 | Package contains no class files. |
| TYA-E0854 | Package directory name is invalid. |
| TYA-E0855 | Package names conflict. |
| TYA-E0856 | Entry file redefines its module. |
| TYA-E0857 | Import name conflicts with another import or module. |
| TYA-E0858 | Variable is undefined. |
| TYA-E0900 | Runtime or CLI reported a general user-facing failure. |
| TYA-E0901 | Task command failed. |
| TYA-E0902 | `tya.toml` could not be found for a project command. |
| TYA-E0903 | Task dependency graph is invalid. |
| TYA-E0904 | Task command form is invalid. |
| TYA-E0905 | Task watch mode failed. |
| TYA-E0906 | Manifest dependency form is invalid. |
| TYA-E0907 | Package dependency operation failed. |
| TYA-E0908 | Package dependency source is invalid. |
| TYA-E0910 | Project name is invalid. |
| TYA-E0911 | Target project directory already exists. |
| TYA-E0912 | Project template is invalid. |
| TYA-E0913 | `--here` was combined with a target name. |
| TYA-E0914 | Native package scaffolding was requested for a non-library template. |
| TYA-E0920 | Native package source file is missing. |
| TYA-E0921 | Native package header file is missing. |
| TYA-E0922 | Native package include directory is missing. |
| TYA-E0923 | Native function is declared by multiple packages. |
| TYA-E0924 | Native package requires missing `pkg-config`. |
| TYA-E0925 | Native package requires a missing `pkg-config` dependency. |
| TYA-E0930 | LSP startup or argument handling failed. |
| TYA-E0931 | LSP I/O or framing failed. |
| TYA-E0933 | LSP rename or request validation failed. |
| TYA-E0940 | Package tool command requires a project manifest. |
| TYA-E0941 | Lockfile is missing, unreadable, or stale. |
| TYA-E0942 | Locked dependency is unavailable locally. |
| TYA-E0943 | Locked dependency content hash mismatched. |
| TYA-E0944 | Requested package tool is not declared. |
| TYA-E0945 | Package tool command failed. |
| TYA-E0946 | One-shot package tool source is invalid. |
| TYA-E0947 | Package tool entry point is invalid. |
