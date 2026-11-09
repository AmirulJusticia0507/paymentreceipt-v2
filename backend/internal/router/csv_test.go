package router

import "testing"

func TestCSVSafeText_PreventsFormulaExecution(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{name: "formula", input: "=1+1", want: "'=1+1"},
		{name: "plus formula", input: "  +cmd", want: "'  +cmd"},
		{name: "minus text", input: "-formula", want: "'-formula"},
		{name: "plain text", input: "General Affairs", want: "General Affairs"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := csvSafeText(test.input); got != test.want {
				t.Errorf("csvSafeText(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
