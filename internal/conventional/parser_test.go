package conventional

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		body    string
		want    Commit
	}{
		{
			name:    "simple feat",
			subject: "feat: add facial attendance",
			want:    Commit{Type: "feat", Description: "add facial attendance"},
		},
		{
			name:    "fix with scope",
			subject: "fix(kiosk): header alignment",
			want:    Commit{Type: "fix", Scope: "kiosk", Description: "header alignment"},
		},
		{
			name:    "breaking with bang",
			subject: "feat!: drop node 16 support",
			want:    Commit{Type: "feat", Description: "drop node 16 support", Breaking: true},
		},
		{
			name:    "breaking with scope and bang",
			subject: "refactor(api)!: rename endpoints",
			want:    Commit{Type: "refactor", Scope: "api", Description: "rename endpoints", Breaking: true},
		},
		{
			name:    "breaking via footer",
			subject: "chore: bump deps",
			body:    "Some context\n\nBREAKING CHANGE: config format changed",
			want:    Commit{Type: "chore", Description: "bump deps", Breaking: true, Body: "Some context\n\nBREAKING CHANGE: config format changed"},
		},
		{
			name:    "non conventional",
			subject: "updated readme",
			want:    Commit{Description: "updated readme"},
		},
		{
			name:    "colon but no type",
			subject: "WIP: still working",
			want:    Commit{Type: "wip", Description: "still working"},
		},
		{
			name:    "url in subject is not a type",
			subject: "see https://example.com for details",
			want:    Commit{Description: "see https://example.com for details"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse("abc123", tt.subject, tt.body)
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Scope != tt.want.Scope {
				t.Errorf("Scope = %q, want %q", got.Scope, tt.want.Scope)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}
			if got.Breaking != tt.want.Breaking {
				t.Errorf("Breaking = %v, want %v", got.Breaking, tt.want.Breaking)
			}
			if got.Hash != "abc123" {
				t.Errorf("Hash = %q, want abc123", got.Hash)
			}
		})
	}
}

func TestIsConventional(t *testing.T) {
	if !Parse("h", "feat: x", "").IsConventional() {
		t.Error("expected conventional")
	}
	if Parse("h", "random text", "").IsConventional() {
		t.Error("expected non-conventional")
	}
}
