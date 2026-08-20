package handler

import "testing"

func TestResolveActivityPageUsesPageSizeWhenLimitMissing(t *testing.T) {
	page, pageSize, offset := resolveActivityPage(1, 20, 0, 0, false)
	if page != 1 || pageSize != 20 || offset != 0 {
		t.Fatalf("got page=%d pageSize=%d offset=%d", page, pageSize, offset)
	}
	page, pageSize, offset = resolveActivityPage(3, 20, 0, 0, false)
	if page != 3 || pageSize != 20 || offset != 40 {
		t.Fatalf("got page=%d pageSize=%d offset=%d", page, pageSize, offset)
	}
}

func TestPaginateActivitiesExposesHasNext(t *testing.T) {
	result := paginateActivities(map[string]any{"items": []any{"a"}, "total": 40}, 1, 20)
	if result["has_next"] != true {
		t.Fatalf("expected has_next, got %#v", result)
	}
	if result["total_items"] != 40 || result["page_size"] != 20 {
		t.Fatalf("unexpected pagination envelope: %#v", result)
	}
}
