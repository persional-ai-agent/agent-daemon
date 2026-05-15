package tools

import "testing"

func TestChannelDirectoryUpsertAndList(t *testing.T) {
	workdir := t.TempDir()
	if err := UpsertChannelDirectory(workdir, ChannelDirectoryEntry{
		Platform: "telegram",
		ChatID:   "100",
		UserID:   "u1",
		UserName: "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertChannelDirectory(workdir, ChannelDirectoryEntry{
		Platform:   "telegram",
		ChatID:     "100",
		HomeTarget: "100",
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := ListChannelDirectory(workdir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d want=1", len(rows))
	}
	if rows[0].Platform != "telegram" || rows[0].ChatID != "100" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	if rows[0].UserID != "u1" || rows[0].HomeTarget != "100" {
		t.Fatalf("unexpected merged row: %+v", rows[0])
	}
}
