package utils

import (
	"math/big"
	"testing"
)

func TestToEther(t *testing.T) {
	type args struct {
		b *big.Int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"case 1", args{big.NewInt(1e18)}, 1},
		{"case 2", args{big.NewInt(1e17)}, 0.1},
		{"case 3", args{big.NewInt(1e16)}, 0.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToEther(tt.args.b); got != tt.want {
				t.Errorf("ToEther() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromEther(t *testing.T) {
	type args struct {
		b float64
	}
	tests := []struct {
		name string
		args args
		want *big.Int
	}{
		{"case 1", args{1}, big.NewInt(1e18)},
		{"case 2", args{0.1}, big.NewInt(1e17)},
		{"case 3", args{0.01}, big.NewInt(1e16)},
		{"case 4", args{0.00100}, big.NewInt(1e15)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToWei(tt.args.b); got.Cmp(tt.want) != 0 {
				t.Errorf("FromEther() = %v, want %v", got, tt.want)
			}
		})
	}
}
