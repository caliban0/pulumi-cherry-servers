package provider

import (
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// autoname determines the name for an auto-named field.
// Priority:
// 1. User provided arg.
// 2. Value from previous inputs.
// 3. A generated name, with a prefix.
func autoname(arg, pref string, old property.Value) (string, error) {
	const (
		randLen = 6  // How many random characters to add to a generated name.
		maxLen  = 28 // Maximum length of a generated name.
	)

	if arg != "" {
		return arg, nil
	}

	if old.IsString() && old.AsString() != "" {
		return old.AsString(), nil
	}

	return resource.NewUniqueHex(pref+"-", randLen, maxLen)
}
