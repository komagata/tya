package runner

import "testing"

func TestValidateFileName(t *testing.T) {
	valid := []string{"main.tya", "user_utils.tya", "string_tools.tya"}
	for _, name := range valid {
		if err := ValidateFileName(name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	invalid := []string{"UserUtils.tya", "user-utils.tya", "userUtils.tya", "main.txt"}
	for _, name := range invalid {
		if err := ValidateFileName(name); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}
