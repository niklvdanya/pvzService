package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginate(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3, 4, 5, 6, 7}

	tests := []struct {
		name      string
		page, per uint64
		want      []int
	}{
		{
			name: "ZeroPerPage_ReturnsEmpty",
			page: 1, per: 0,
			want: []int{},
		},
		{
			name: "PageZero_TreatedAsFirst",
			page: 0, per: 3,
			want: []int{1, 2, 3},
		},
		{
			name: "FirstPage_FullChunk",
			page: 1, per: 4,
			want: []int{1, 2, 3, 4},
		},
		{
			name: "SecondPage_PartialChunk",
			page: 2, per: 4,
			want: []int{5, 6, 7},
		},
		{
			name: "PageBeyondRange_ReturnsEmpty",
			page: 3, per: 4,
			want: []int{},
		},
		{
			name: "ExactDivision_LastPageFull",
			page: 2, per: 3,
			want: []int{4, 5, 6},
		},
		{
			name: "ExactDivision_PageOutOfRange",
			page: 3, per: 3,
			want: []int{7},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Paginate(items, tc.page, tc.per)
			assert.Equal(t, tc.want, got)
		})
	}
}
