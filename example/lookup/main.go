package main

import (
	"fmt"

	"github.com/oddinpay/aaguid"
)

func main() {
	target := "54d9fee8-e621-4291-8b18-7157b99c5bec"
	id := aaguid.MustParseAAGUID(target)

	if info, ok := aaguid.PasskeyAuthenticatorAAGUIDs[id]; ok {
		fmt.Printf("[FOUND] %s -> %s\n", target, info.Name)
	} else if info, ok := aaguid.MetadataAAGUIDs[id]; ok {
		fmt.Printf("[FOUND] %s -> %s\n", target, info.Name)
	} else {
		fmt.Println("Could not find:", target)
	}
}
