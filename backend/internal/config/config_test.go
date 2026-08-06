package config

import "testing"

func TestValidateAccessSecret(t *testing.T) {
	ok := []string{"test123456", "abcdefgh", "a1234567", "Ab1!xxxx"}
	for _, s := range ok {
		if err := ValidateAccessSecret(s); err != nil {
			t.Fatalf("%q should pass: %v", s, err)
		}
	}
	bad := []struct {
		secret string
		sub    string
	}{
		{"", "空"},
		{"   ", "空"},
		{"1234567", "长度"},
		{"abcdefg", "长度"},
		{"12345678", "纯数字"},
		{"00000000", "纯数字"},
		{"123456789012", "纯数字"},
	}
	for _, tc := range bad {
		err := ValidateAccessSecret(tc.secret)
		if err == nil {
			t.Fatalf("%q should fail (%s)", tc.secret, tc.sub)
		}
	}
}
