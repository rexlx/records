package main

import (
	"testing"
)

func TestSanitizeServiceName(t *testing.T) {
	type test struct {
		input    string
		expected string
	}
	tests := []test{
		{input: "test service", expected: "test_service"},
		{input: "Best Service", expected: "Best_Service"},
		{input: "another_service", expected: "another_service"},
		{input: "broke\tservice", expected: "broke	service"},
	}
	for _, tc := range tests {
		output := SanitizeServiceName(tc.input)
		if output != tc.expected {
			t.Errorf("expected %v, got %v", tc.expected, output)
		}
	}
}
