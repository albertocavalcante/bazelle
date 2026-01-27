package kotlin

import "testing"

func TestStripComments(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		inBlock bool
		want    string
		wantIn  bool
	}{
		{
			name:    "no comments",
			line:    "val x = 1",
			inBlock: false,
			want:    "val x = 1",
			wantIn:  false,
		},
		{
			name:    "line comment",
			line:    "val x = 1 // comment",
			inBlock: false,
			want:    "val x = 1 ",
			wantIn:  false,
		},
		{
			name:    "block comment inline",
			line:    "val x = /* comment */ 1",
			inBlock: false,
			want:    "val x =  1",
			wantIn:  false,
		},
		{
			name:    "block comment start",
			line:    "val x = /* comment",
			inBlock: false,
			want:    "val x = ",
			wantIn:  true,
		},
		{
			name:    "inside block comment",
			line:    "still in comment",
			inBlock: true,
			want:    "",
			wantIn:  true,
		},
		{
			name:    "block comment end",
			line:    "end of comment */ val y = 2",
			inBlock: true,
			want:    " val y = 2",
			wantIn:  false,
		},
		// String handling tests
		{
			name:    "URL in string preserved",
			line:    `val url = "https://example.com"`,
			inBlock: false,
			want:    `val url = "https://example.com"`,
			wantIn:  false,
		},
		{
			name:    "URL in string with trailing comment",
			line:    `val url = "https://example.com" // comment`,
			inBlock: false,
			want:    `val url = "https://example.com" `,
			wantIn:  false,
		},
		{
			name:    "block comment syntax in string",
			line:    `val x = "/* not a comment */"`,
			inBlock: false,
			want:    `val x = "/* not a comment */"`,
			wantIn:  false,
		},
		{
			name:    "line comment syntax in string",
			line:    `val x = "// not a comment"`,
			inBlock: false,
			want:    `val x = "// not a comment"`,
			wantIn:  false,
		},
		{
			name:    "escaped quote in string",
			line:    `val x = "he said \"hello\"" // comment`,
			inBlock: false,
			want:    `val x = "he said \"hello\"" `,
			wantIn:  false,
		},
		{
			name:    "multiple strings",
			line:    `val x = "a" + "b" // comment`,
			inBlock: false,
			want:    `val x = "a" + "b" `,
			wantIn:  false,
		},
		{
			name:    "empty string",
			line:    `val x = "" // comment`,
			inBlock: false,
			want:    `val x = "" `,
			wantIn:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotIn := stripComments(tt.line, tt.inBlock)
			if got != tt.want {
				t.Errorf("stripComments() got = %q, want %q", got, tt.want)
			}
			if gotIn != tt.wantIn {
				t.Errorf("stripComments() inBlock = %v, want %v", gotIn, tt.wantIn)
			}
		})
	}
}
