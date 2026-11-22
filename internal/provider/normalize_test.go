package provider

import "testing"

func TestSameName(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		returned   string
		want       bool
	}{
		{"identical", "example.com", "example.com", true},
		{"case only", "Example.COM", "example.com", true},
		{"trailing dot", "example.com.", "example.com", true},
		{"case and dot", "Example.Com.", "example.com", true},
		{"idn to punycode", "bücher.example", "xn--bcher-kva.example", true},
		{"punycode already", "xn--bcher-kva.example", "xn--bcher-kva.example", true},
		{"different names", "example.com", "example.net", false},
		{"subdomain is not the apex", "www.example.com", "example.com", false},
		{"empty configured", "", "example.com", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameName(tc.configured, tc.returned); got != tc.want {
				t.Errorf("sameName(%q, %q) = %v, want %v",
					tc.configured, tc.returned, got, tc.want)
			}
		})
	}
}

// Every spelling the API normalises must survive as written.
func TestPreserveConfiguredKeepsAnEquivalentSpelling(t *testing.T) {
	cases := []struct {
		configured string
		returned   string
		want       string
	}{
		{"Example.com", "example.com", "Example.com"},
		{"example.com.", "example.com", "example.com."},
		{"bücher.example", "xn--bcher-kva.example", "bücher.example"},
		// A genuine change elsewhere must still surface as drift.
		{"example.com", "other.com", "other.com"},
		// Nothing configured yet: take what the API says.
		{"", "example.com", "example.com"},
	}

	for _, tc := range cases {
		if got := preserveConfigured(tc.configured, tc.returned); got != tc.want {
			t.Errorf("preserveConfigured(%q, %q) = %q, want %q",
				tc.configured, tc.returned, got, tc.want)
		}
	}
}
