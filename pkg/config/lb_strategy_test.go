package config

import "testing"

func TestRoundRobinStartsAtFirstAndCycles(t *testing.T) {
	key := t.Name()
	want := []int{0, 1, 2, 0}
	for index, expected := range want {
		if got := GetLBIndex("round_robin", key, 3); got != expected {
			t.Fatalf("call %d returned %d, want %d", index+1, got, expected)
		}
	}
}

func TestHashStrategyIsStable(t *testing.T) {
	first := GetLBIndex("hash", "stable-route", 17)
	for range 20 {
		if got := GetLBIndex("hash", "stable-route", 17); got != first {
			t.Fatalf("hash index changed from %d to %d", first, got)
		}
	}
}
