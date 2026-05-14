package main

type lintRuleInfo struct {
	Title  string
	DocURL string
}

func lintRuleMetadata(code string) lintRuleInfo {
	switch code {
	case "TYAL0001":
		return lintRuleInfo{"Unused local", "https://tya-lang.org/lint.html#tyal0001"}
	case "TYAL0002":
		return lintRuleInfo{"Dead code after return or raise", "https://tya-lang.org/lint.html#tyal0002"}
	case "TYAL0003":
		return lintRuleInfo{"Redundant constant if", "https://tya-lang.org/lint.html#tyal0003"}
	case "TYAL0004":
		return lintRuleInfo{"Deeply nested block", "https://tya-lang.org/lint.html#tyal0004"}
	case "TYAL0005":
		return lintRuleInfo{"Long function body", "https://tya-lang.org/lint.html#tyal0005"}
	case "TYAL0006":
		return lintRuleInfo{"Suspicious for index pattern", "https://tya-lang.org/lint.html#tyal0006"}
	case "TYAL0007":
		return lintRuleInfo{"Unused function parameter", "https://tya-lang.org/lint.html#tyal0007"}
	case "TYAL0008":
		return lintRuleInfo{"Shadowed binding", "https://tya-lang.org/lint.html#tyal0008"}
	default:
		return lintRuleInfo{"Lint finding", "https://tya-lang.org/lint.html"}
	}
}

func enrichLintFinding(f lintFinding) lintFinding {
	info := lintRuleMetadata(f.Code)
	f.Title = info.Title
	f.DocURL = info.DocURL
	return f
}
