package knowledge

import "testing"

func TestSanitizeFTS5Query(t *testing.T) {
	if g := SanitizeFTS5Query("hello world"); g == "" {
		t.Fatal("want non-empty")
	}
	if g := SanitizeFTS5Query(`foo "bar"`); g == "" {
		t.Fatal("want non-empty")
	}
	if g := SanitizeFTS5Query("   "); g != "" {
		t.Fatalf("want empty got %q", g)
	}
}
