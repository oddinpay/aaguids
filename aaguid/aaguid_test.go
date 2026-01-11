package aaguid

import "testing"

func TestSpecificAAGUIDs(t *testing.T) {
	targets := []string{
		"54d9fee8-e621-4291-8b18-7157b99c5bec",
		"560a780cb6ae4f03b110082f856425b4",
	}

	// Put all your data sources here
	allMaps := []map[AAGUID]string{
		PasskeyAuthenticatorAAGUIDs,
		MetadataAAGUIDs,
	}

	for _, target := range targets {
		id := MustParseAAGUID(target)
		found := false

		// Check all maps in one loop
		for _, m := range allMaps {
			if name, ok := m[id]; ok {
				t.Logf("[FOUND] %s -> %s", target, name)
				found = true
			}
		}

		if !found {
			t.Errorf("Could not find: %s", target)
		}
	}
}
