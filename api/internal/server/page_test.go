package server

import (
	"net/http/httptest"
	"testing"
)

func TestReadPage(t *testing.T) {
	tests := map[string]pageRequest{
		"":                      {0, 50},
		"?offset=50":            {50, 50},
		"?offset=100&limit=20":  {100, 20},
		"?limit=500":            {0, 100},
		"?offset=-5&limit=0":    {0, 50},
		"?offset=abc&limit=xyz": {0, 50},
	}
	for query, want := range tests {
		got := readPage(httptest.NewRequest("GET", "/api/x"+query, nil))
		if got != want {
			t.Errorf("%q: got %+v, want %+v", query, got, want)
		}
	}
	if got := (pageRequest{offset: 50, limit: 50}).sql(4); got != " LIMIT $4 OFFSET $5" {
		t.Errorf("sql = %q", got)
	}
}
