package service

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Teresa's Eatery", "teresa-s-eatery"},
		{"  My   Cool  Cafe  ", "my-cool-cafe"},
		{"UPPER-case", "upper-case"},
		{"café münchen", "caf-m-nchen"},
		{"---", ""},
		{"a", "a"},
		{"123 shop", "123-shop"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := slugify(tt.in); got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
