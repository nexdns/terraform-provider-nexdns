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
		{"idn to punycode", "педсайт.рф", "xn--80ahcs0aoj.xn--p1ai", true},
		{"punycode already", "xn--80ahcs0aoj.xn--p1ai", "xn--80ahcs0aoj.xn--p1ai", true},
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

// The apply fails with "Provider produced inconsistent result after apply" when
// what is written to state differs from what the plan held - and by then the
// change has already been made on the server. Every spelling the platform
// normalises has to survive as the practitioner wrote it.
func TestPreserveConfiguredKeepsAnEquivalentSpelling(t *testing.T) {
	cases := []struct {
		configured string
		returned   string
		want       string
	}{
		{"Example.com", "example.com", "Example.com"},
		{"example.com.", "example.com", "example.com."},
		{"педсайт.рф", "xn--80ahcs0aoj.xn--p1ai", "педсайт.рф"},
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
