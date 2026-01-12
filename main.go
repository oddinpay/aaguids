package main

import (
	"fmt"

	"github.com/oddinpay/aaguid/aaguid"
)

func main() {
	targets := []string{
		"54d9fee8-e621-4291-8b18-7157b99c5bec",
		"560a780cb6ae4f03b110082f856425b4",
	}

	// Put all your data sources here
	allMaps := []map[aaguid.AAGUID]aaguid.AAGUIDInfo{
		aaguid.PasskeyAuthenticatorAAGUIDs,
		aaguid.MetadataAAGUIDs,
	}

	for _, target := range targets {
		id := aaguid.MustParseAAGUID(target)
		found := false

		// Check all maps in one loop
		for _, m := range allMaps {
			if name, ok := m[id]; ok {
				fmt.Println("[FOUND]", target, "->", name)
				found = true
			}
		}

		if !found {
			fmt.Println("Could not find:", target)
		}
	}
}
