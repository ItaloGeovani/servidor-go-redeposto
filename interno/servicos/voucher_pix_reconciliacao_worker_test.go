package servicos

import "testing"

func TestStatusPixIndicaEstornoTotal(t *testing.T) {
	casos := []struct {
		st   string
		want bool
	}{
		{"refunded", true},
		{"REFUNDED", true},
		{"cancelled", true},
		{"canceled", true},
		{"charged_back", true},
		{"approved", false},
		{"pending", false},
		{"", false},
		{"  refunded  ", true},
	}
	for _, c := range casos {
		if got := statusPixIndicaEstornoTotal(c.st); got != c.want {
			t.Fatalf("statusPixIndicaEstornoTotal(%q)=%v want %v", c.st, got, c.want)
		}
	}
}
