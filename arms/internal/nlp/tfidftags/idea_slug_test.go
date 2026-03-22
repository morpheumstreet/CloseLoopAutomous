package tfidftags

import "testing"

func TestSlugPrefixFromTags(t *testing.T) {
	tags := []TagScore{
		{Token: "oauth", Score: 1},
		{Token: "API-v2", Score: 0.9},
		{Token: "x", Score: 0.8},
	}
	got := SlugPrefixFromTags(tags, 4, 48)
	if got != "oauth-apiv2" {
		t.Fatalf("got %q want oauth-apiv2", got)
	}
}

func TestSlugPrefixFromTagsTruncateRunes(t *testing.T) {
	tags := []TagScore{
		{Token: "abcdefghij", Score: 1},
		{Token: "klmnopqrs", Score: 0.9},
	}
	got := SlugPrefixFromTags(tags, 4, 12)
	if len(got) > 12 {
		t.Fatalf("len %d > 12: %q", len(got), got)
	}
}

func TestSlugPrefixFromTagsEmpty(t *testing.T) {
	if s := SlugPrefixFromTags(nil, 4, 48); s != "" {
		t.Fatalf("want empty, got %q", s)
	}
}
