package mtproto

import "testing"

func TestDefaultDCOptions(t *testing.T) {
	if len(DefaultDCOptions) < 5 {
		t.Fatalf("DefaultDCOptions length = %d, want at least 5", len(DefaultDCOptions))
	}
	for _, dc := range DefaultDCOptions {
		if dc.ID == 0 || dc.Address == "" {
			t.Fatalf("invalid dc option: %+v", dc)
		}
	}
}
