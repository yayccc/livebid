package idgen

import "testing"

func TestGeneratorNext(t *testing.T) {
	g := New(1)
	first := g.Next()
	second := g.Next()

	if first <= 0 {
		t.Fatalf("expected positive id, got %d", first)
	}
	if second <= first {
		t.Fatalf("expected increasing ids, got first=%d second=%d", first, second)
	}
}
