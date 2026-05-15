package gateway

import "testing"

func TestIdentityStoreBindAndResolve(t *testing.T) {
	s := newIdentityStore(t.TempDir())
	if err := s.bind("telegram", "u1", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := s.bind("telegram", "u1", "alice-2"); err != nil {
		t.Fatal(err)
	}
	got, err := s.resolve("telegram", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "alice-2" {
		t.Fatalf("global id=%q want=alice-2", got)
	}
	if err := s.unbind("telegram", "u1"); err != nil {
		t.Fatal(err)
	}
	got, err = s.resolve("telegram", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("global id after unbind=%q want empty", got)
	}
}
