package provider

import (
	"strings"

	"golang.org/x/net/idna"
)

// sameName reports whether two DNS names are the same, ignoring case, a
// trailing dot and IDN form. The API stores names lowercased, without the
// dot and in punycode, and Terraform fails the apply when state differs from
// the configured value.
func sameName(configured, returned string) bool {
	configured = strings.TrimSuffix(configured, ".")
	returned = strings.TrimSuffix(returned, ".")

	if strings.EqualFold(configured, returned) {
		return true
	}

	// ToASCII, not the Lookup profile, so punycode input compares equal to itself.
	ascii, err := idna.ToASCII(configured)
	if err != nil {
		return false
	}

	return strings.EqualFold(ascii, returned)
}

// preserveConfigured returns the configured spelling when it names the same
// name as the API value, and the API value otherwise, so a real change shows
// as drift.
func preserveConfigured(configured, returned string) string {
	if configured != "" && configured != returned && sameName(configured, returned) {
		return configured
	}

	return returned
}
