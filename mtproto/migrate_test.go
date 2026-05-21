package mtproto

import "testing"

func TestMigrationDCID(t *testing.T) {
	tests := []struct {
		message string
		want    int
		ok      bool
	}{
		{"USER_MIGRATE_5", 5, true},
		{"PHONE_MIGRATE_4", 4, true},
		{"NETWORK_MIGRATE_2", 2, true},
		{"FILE_MIGRATE_3", 3, true},
		{"AUTH_KEY_UNREGISTERED", 0, false},
	}
	for _, tt := range tests {
		got, ok := MigrationDCID(&RPCError{Code: 303, Message: tt.message})
		if got != tt.want || ok != tt.ok {
			t.Fatalf("MigrationDCID(%q) = (%d,%v), want (%d,%v)", tt.message, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDCOptionsByID(t *testing.T) {
	got := dcOptionsByID(5)
	if len(got) == 0 {
		t.Fatal("expected DC 5 options")
	}
	for _, dc := range got {
		if dc.ID != 5 {
			t.Fatalf("unexpected dc id %d", dc.ID)
		}
	}
}
