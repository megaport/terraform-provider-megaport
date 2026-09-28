package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	megaport "github.com/megaport/megaportgo"
)

func TestNATGatewaySpeedSessionSupported(t *testing.T) {
	t.Parallel()

	// Synthetic matrix; values are illustrative, not real availability data.
	matrix := []*megaport.NATGatewaySession{
		{SpeedMbps: 1000, SessionCount: []int{16000, 32000}},
		{SpeedMbps: 2000, SessionCount: []int{32000, 64000}},
		nil, // must be skipped without panicking
		{SpeedMbps: 5000, SessionCount: []int{128000}},
	}

	cases := []struct {
		name            string
		matrix          []*megaport.NATGatewaySession
		speed           int
		sessionCount    int
		wantOK          bool
		wantPath        path.Path // checked only when wantOK is false
		wantMsgContains []string
	}{
		{
			name:         "valid speed and session",
			matrix:       matrix,
			speed:        1000,
			sessionCount: 32000,
			wantOK:       true,
		},
		{
			name:         "nil entries are skipped",
			matrix:       matrix,
			speed:        5000,
			sessionCount: 128000,
			wantOK:       true,
		},
		{
			name:            "unsupported speed lists every supported speed",
			matrix:          matrix,
			speed:           500,
			sessionCount:    16000,
			wantOK:          false,
			wantPath:        path.Root("speed"),
			wantMsgContains: []string{"500", "1000", "2000", "5000"},
		},
		{
			name:            "supported speed, unsupported session",
			matrix:          matrix,
			speed:           1000,
			sessionCount:    99999,
			wantOK:          false,
			wantPath:        path.Root("session_count"),
			wantMsgContains: []string{"99999", "1000", "16000", "32000"},
		},
		{
			name:         "empty matrix rejects on speed",
			matrix:       []*megaport.NATGatewaySession{},
			speed:        1000,
			sessionCount: 16000,
			wantOK:       false,
			wantPath:     path.Root("speed"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotPath, gotMsg, gotOK := natGatewaySpeedSessionSupported(tc.matrix, tc.speed, tc.sessionCount)

			if gotOK != tc.wantOK {
				t.Fatalf("ok = %v, want %v (msg=%q)", gotOK, tc.wantOK, gotMsg)
			}
			if tc.wantOK {
				if gotMsg != "" {
					t.Errorf("expected empty message on success, got %q", gotMsg)
				}
				if !gotPath.Equal(path.Empty()) {
					t.Errorf("expected empty path on success, got %s", gotPath)
				}
				return
			}
			if !gotPath.Equal(tc.wantPath) {
				t.Errorf("path = %s, want %s", gotPath, tc.wantPath)
			}
			if gotMsg == "" {
				t.Error("expected a non-empty diagnostic message on failure")
			}
			for _, sub := range tc.wantMsgContains {
				if !strings.Contains(gotMsg, sub) {
					t.Errorf("message %q does not contain %q", gotMsg, sub)
				}
			}
		})
	}
}
