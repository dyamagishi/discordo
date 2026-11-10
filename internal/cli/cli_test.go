package cli

import (
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantArgs   []string
		wantLimit  uint
		wantBefore string
		wantJSON   bool
	}{
		{
			name:      "flags before positional",
			args:      []string{"-limit", "5", "123"},
			wantArgs:  []string{"123"},
			wantLimit: 5,
		},
		{
			name:      "flags after positional",
			args:      []string{"123", "-limit", "7"},
			wantArgs:  []string{"123"},
			wantLimit: 7,
		},
		{
			name:       "mixed",
			args:       []string{"123", "-before", "456", "789"},
			wantArgs:   []string{"123", "789"},
			wantBefore: "456",
		},
		{
			name:      "equals form",
			args:      []string{"-limit=3", "123"},
			wantArgs:  []string{"123"},
			wantLimit: 3,
		},
		{
			name:     "bool flag",
			args:     []string{"-json", "123"},
			wantArgs: []string{"123"},
			wantJSON: true,
		},
		{
			name:     "terminator",
			args:     []string{"--", "-not-a-flag"},
			wantArgs: []string{"-not-a-flag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := newFlagSet("test")
			limit := fs.Uint("limit", 0, "")
			before := fs.String("before", "", "")
			jsonOut := fs.Bool("json", false, "")

			args, err := parseFlags(fs, tt.args)
			if err != nil {
				t.Fatalf("parseFlags() error = %v", err)
			}

			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", args, tt.wantArgs)
			}
			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Fatalf("args = %v, want %v", args, tt.wantArgs)
				}
			}

			if *limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", *limit, tt.wantLimit)
			}
			if *before != tt.wantBefore {
				t.Errorf("before = %q, want %q", *before, tt.wantBefore)
			}
			if *jsonOut != tt.wantJSON {
				t.Errorf("json = %v, want %v", *jsonOut, tt.wantJSON)
			}
		})
	}
}

func TestParseSnowflake(t *testing.T) {
	if _, err := parseSnowflake("123456789012345678", "id"); err != nil {
		t.Fatalf("valid snowflake rejected: %v", err)
	}

	for _, invalid := range []string{"", "0", "abc", "-1", "1.5"} {
		if _, err := parseSnowflake(invalid, "id"); err == nil {
			t.Errorf("parseSnowflake(%q) = nil, want error", invalid)
		}
	}
}
