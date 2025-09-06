package main

import (
	"fmt"
	"testing"
)

func TestIsPrime(t *testing.T) {
	tests := []struct {
		n    int
		want bool
	}{
		// 境界値
		{0, false},
		{1, false},
		{2, true},
		{3, true},
		// 小さい素数
		{5, true},
		{7, true},
		{11, true},
		{13, true},
		{17, true},
		{19, true},
		{23, true},
		// 合成数
		{4, false},
		{6, false},
		{8, false},
		{9, false},
		{10, false},
		{12, false},
		{15, false},
		{20, false},
		// 大きい数
		{97, true},
		{100, false},
		{1009, true},
		{1024, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.n), func(t *testing.T) {
			if got := isPrime(tt.n); got != tt.want {
				t.Errorf("isPrime(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}
