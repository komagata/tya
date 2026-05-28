;;; tya-mode.el --- Major mode for Tya -*- lexical-binding: t; -*-

;; Version: 0.61.0
;; Package-Requires: ((emacs "28.1"))
;; Keywords: languages
;; URL: https://github.com/komagata/tya

;;; Commentary:

;; Syntax coloring and indentation for the Tya programming language.

;;; Code:

(defgroup tya nil
  "Major mode for Tya."
  :group 'languages)

(defconst tya-mode-keywords
  '("if" "elseif" "else" "while" "for" "in" "break" "continue" "return"
    "raise" "try" "catch" "match" "case" "when" "select" "receive" "send"
    "timeout" "default"))

(defconst tya-mode-declarations
  '("class" "module" "interface" "struct" "record" "implements" "extends" "abstract" "final"
    "private" "static" "initialize" "import" "as"))

(defconst tya-mode-concurrency
  '("spawn" "await" "scope"))

(defconst tya-mode-font-lock-keywords
  `((,(regexp-opt tya-mode-keywords 'symbols) . font-lock-keyword-face)
    (,(regexp-opt tya-mode-declarations 'symbols) . font-lock-type-face)
    (,(regexp-opt tya-mode-concurrency 'symbols) . font-lock-keyword-face)
    (,(regexp-opt '("true" "false" "nil") 'symbols) . font-lock-constant-face)
    (,(regexp-opt '("self" "Self" "super") 'symbols) . font-lock-variable-name-face)
    (,(regexp-opt '("and" "or" "not") 'symbols) . font-lock-builtin-face)
    ("\\_<\\(class\\|interface\\|struct\\|record\\)\\_>[ \t]+\\([A-Z][A-Za-z0-9_]*\\)"
     (2 font-lock-type-face))
    ("\\_<module\\_>[ \t]+\\([A-Za-z_][A-Za-z0-9_]*\\)"
     (1 font-lock-constant-face))
    ("\\_<\\([A-Za-z_][A-Za-z0-9_?]*\\)\\_>[ \t]*=[ \t]*->"
     (1 font-lock-function-name-face))
    ("\\_<0x[0-9a-fA-F_]+\\_>" . font-lock-constant-face)
    ("\\_<0b[01_]+\\_>" . font-lock-constant-face)
    ("\\_<[0-9][0-9_]*\\(\\.[0-9][0-9_]*\\)?\\_>" . font-lock-constant-face)))

(defvar tya-mode-syntax-table
  (let ((table (make-syntax-table)))
    (modify-syntax-entry ?# "<" table)
    (modify-syntax-entry ?\n ">" table)
    (modify-syntax-entry ?_ "w" table)
    table)
  "Syntax table for `tya-mode'.")

;;;###autoload
(define-derived-mode tya-mode prog-mode "Tya"
  "Major mode for editing Tya source."
  :syntax-table tya-mode-syntax-table
  (setq-local comment-start "#")
  (setq-local comment-end "")
  (setq-local indent-tabs-mode nil)
  (setq-local tab-width 2)
  (setq-local font-lock-defaults '(tya-mode-font-lock-keywords)))

;;;###autoload
(add-to-list 'auto-mode-alist '("\\.tya\\'" . tya-mode))

(provide 'tya-mode)

;;; tya-mode.el ends here
