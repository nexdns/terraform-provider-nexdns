package provider

import (
	"strings"

	"golang.org/x/net/idna"
)

// sameName reports whether two DNS names denote the same name, ignoring the
// spellings the platform normalises away: letter case, a trailing dot, and the
// Unicode form of an internationalised name.
//
// The API stores a name lowercased, without the trailing dot and in its ASCII
// (punycode) form. Terraform, meanwhile, requires that what a resource writes to
// state equals what the plan said, so writing the normalised form back over a
// configuration that spelled the name any other way ends the apply with
// "Provider produced inconsistent result after apply" - after the change has
// already been made on the server, which is the worst moment to fail.
func sameName(configured, returned string) bool {
	configured = strings.TrimSuffix(configured, ".")
	returned = strings.TrimSuffix(returned, ".")

	if strings.EqualFold(configured, returned) {
		return true
	}

	// idna.ToASCII is deliberately the lookup profile's lenient sibling: a
	// configuration that already holds punycode must compare equal to itself,
	// and a name that will not convert is simply not equivalent.
	ascii, err := idna.ToASCII(configured)
	if err != nil {
		return false
	}

	return strings.EqualFold(ascii, returned)
}

// preserveConfigured returns the spelling the practitioner wrote when it denotes
// the same name the API returned, and the API's own value otherwise (a genuine
// change made elsewhere, which Terraform should show as drift).
//
// Pass the value currently in the model - on create that is the configuration,
// on refresh the prior state, and both are spellings the practitioner chose.
func preserveConfigured(configured, returned string) string {
	if configured != "" && configured != returned && sameName(configured, returned) {
		return configured
	}

	return returned
}
