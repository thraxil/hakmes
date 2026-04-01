package main

import "testing"

func TestKeyFromString(t *testing.T) {
	cases := []struct {
		input   string
		valid   bool
		algo    string
		val     string
		wantErr bool
	}{
		{"sha1:1234567890123456789012345678901234567890", true, "sha1", "1234567890123456789012345678901234567890", false},
		{"md5:12345678901234567890123456789012", false, "", "", true},
		{"sha1:123", false, "", "", true},
	}

	for _, c := range cases {
		k, err := keyFromString(c.input)
		if (err != nil) != c.wantErr {
			t.Errorf("keyFromString(%q) error = %v, wantErr %v", c.input, err, c.wantErr)
			continue
		}
		if err == nil {
			if k.Algorithm != c.algo {
				t.Errorf("keyFromString(%q) algorithm = %q, want %q", c.input, k.Algorithm, c.algo)
			}
			if string(k.Value) != c.val {
				t.Errorf("keyFromString(%q) value = %q, want %q", c.input, string(k.Value), c.val)
			}
		}
	}
}

func TestKeyString(t *testing.T) {
	k := &key{Algorithm: "sha1", Value: []byte("1234567890123456789012345678901234567890")}
	expected := "sha1:1234567890123456789012345678901234567890"
	if k.String() != expected {
		t.Errorf("k.String() = %q, want %q", k.String(), expected)
	}
}

func TestKeyValid(t *testing.T) {
	cases := []struct {
		k     key
		valid bool
	}{
		{key{"sha1", []byte("1234567890123456789012345678901234567890")}, true},
		{key{"md5", []byte("12345678901234567890123456789012")}, false},
		{key{"sha1", []byte("123")}, false},
	}

	for _, c := range cases {
		if c.k.Valid() != c.valid {
			t.Errorf("%v.Valid() = %v, want %v", c.k, c.k.Valid(), c.valid)
		}
	}
}
