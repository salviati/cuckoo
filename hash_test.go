package cuckoo

import (
	"testing"
)

func TestHash(t *testing.T) {
	type args struct {
		k    uint32
		seed uint32
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			"murmur3_32",
			args{
				k:    10,
				seed: 0,
			},
			3675908860,
		},
		{
			"xx_32",
			args{
				k:    10,
				seed: 0,
			},
			2946140445,
		},
		{
			"mem_32",
			args{
				k: 10,
			},
			825698977,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "murmur3_32":
				if got := murmur3_32(tt.args.k, tt.args.seed); got != tt.want {
					t.Errorf("murmur3_32() = %v, want %v", got, tt.want)
				}
			case "xx_32":
				if got := xx_32(tt.args.k, tt.args.seed); got != tt.want {
					t.Errorf("xx_32() = %v, want %v", got, tt.want)
				}
			case "mem_32":
				if got := mem_32(tt.args.k, tt.args.seed); got != tt.want {
					t.Errorf("mem_32() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
