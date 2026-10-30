package utils_test

import (
	"reflect"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestSafeSum(t *testing.T) {
	if got := utils.SafeSum([]int{1, 2, 3}); got != 6 {
		t.Errorf("utils.SafeSum() = %d, want 6", got)
	}
}

func TestUnique(t *testing.T) {
	input := []int{1, 2, 2, 3, 1}
	expected := []int{1, 2, 3}
	got := utils.Unique(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.Unique() = %v, want %v", got, expected)
	}
}
