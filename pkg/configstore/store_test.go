package configstore

import (
	"context"
	"testing"
)

func TestRevisionLifecycle(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()

	first, err := store.CreateRevision(ctx, []byte(`{"server_port":":9090"}`), "import", "first", true)
	if err != nil {
		t.Fatalf("create first revision: %v", err)
	}
	second, err := store.CreateRevision(ctx, []byte(`{"server_port":":9191"}`), "admin", "second", false)
	if err != nil {
		t.Fatalf("create second revision: %v", err)
	}
	active, payload, err := store.Active(ctx)
	if err != nil || active.ID != first.ID || string(payload) != `{"server_port":":9090"}` {
		t.Fatalf("unexpected active revision: %#v %s %v", active, payload, err)
	}
	activated, _, err := store.Activate(ctx, second.ID)
	if err != nil || !activated.Active {
		t.Fatalf("activate second revision: %#v %v", activated, err)
	}
	items, err := store.List(ctx, 10)
	if err != nil || len(items) != 2 || items[0].ID != second.ID || !items[0].Active {
		t.Fatalf("unexpected revisions: %#v %v", items, err)
	}
}
