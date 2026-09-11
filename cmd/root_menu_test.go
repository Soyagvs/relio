package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/soyagvs/relio/internal/guide"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/releases"
)

func TestIsHardQuit(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("x"), false},
		{"pick.ErrQuit", pick.ErrQuit, true},
		{"releases.ErrQuit", releases.ErrQuit, true},
		{"guide.ErrQuit", guide.ErrQuit, true},
		{"wrapped releases.ErrQuit", fmt.Errorf("browser: %w", releases.ErrQuit), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isHardQuit(tc.err); got != tc.want {
				t.Errorf("isHardQuit(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
