package main

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestSepLabel(t *testing.T) {
	type item struct {
		label    string
		expected []string
	}

	table := []item{
		{"2.0.1", []string{"2", "0", "1"}},
		{"v0.0.1", []string{"v", "0", "0", "1"}},
		{"2xFg33.+f.5", []string{"2", "xFg", "33", "f", "5"}},
	}

	for _, it := range table {
		t.Run(it.label, func(t *testing.T) {
			s := sepLabel(it.label)
			if !slices.Equal(s, it.expected) {
				t.Errorf("Expected %v, got %v", it.expected, s)
			}
		})
	}
}

func TestVerCmp(t *testing.T) {
	type item struct {
		v1, v2   string
		expected int
	}

	table := []item{
		{"1.0010", "1.9", 1},
		{"1.05", "1.5", 0},
		{"1.0", "1", 1},
		{"1", "1.0", -1},
		{"2.50", "2.5", 1},
		{"FC5", "fc4", -1},
		{"2a", "2.0", -1},
		{"1.0", "1.fc4", 1},
		{"3.0.0_fc", "3.0.0.fc", 0},
		{"4.1__", "4.1+", 0},
	}

	for _, it := range table {
		t.Run(it.v1+"/"+it.v2, func(t *testing.T) {
			c := vercmp(it.v1, it.v2)
			if c != it.expected {
				t.Errorf("Expected %d, got %d", it.expected, c)
			}

			// Ensure opposite comparison gives opposite value
			c = -vercmp(it.v2, it.v1)
			if c != it.expected {
				t.Errorf("Expected %d, got %d (opposite)", it.expected, c)
			}
		})
	}
}
