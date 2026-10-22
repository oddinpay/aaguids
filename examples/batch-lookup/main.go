package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/oddinpay/aaguids"
)

// --- UI Colors & Constants ---
const (
	boxInnerWidth = 63
	GREEN         = "\033[0;32m"
	RED           = "\033[0;31m"
	YELLOW        = "\033[0;33m"
	BLUE          = "\033[0;34m"
	CYAN          = "\033[0;36m"
	MAGENTA       = "\033[0;35m"
	BOLD          = "\033[1m"
	DIM           = "\033[2m"
	NC            = "\033[0m"
)

type LookupResult struct {
	Target    string
	Found     bool
	Name      string
	IconLight string
	IconDark  string
}

func boxHeader(color, icon, title string) string {
	titleRunes := utf8.RuneCountInString(title)
	dashes := max(boxInnerWidth-(2+1+2+1+titleRunes+1), 1)
	return fmt.Sprintf("%s┌── %s %s %s┐%s", color, icon, title, strings.Repeat("─", dashes), NC)
}

func boxFooter(color string) string {
	return fmt.Sprintf("%s└%s┘%s", color, strings.Repeat("─", boxInnerWidth), NC)
}

func main() {
	targets := []string{
		"54d9fee8-e621-4291-8b18-7157b99c5bec",
		"560a780cb6ae4f03b110082f856425b4",
		"bada5566-a7aa-401f-bd96-45619a55120d",
	}

	allMaps := []map[aaguids.AAGUID]aaguids.AAGUIDInfo{
		aaguids.MetadataAAGUIDs,
	}

	var results []LookupResult
	foundCount := 0

	for _, target := range targets {
		id := aaguids.MustParseAAGUID(target)
		found := false
		var info aaguids.AAGUIDInfo

		for _, m := range allMaps {
			if val, ok := m[id]; ok {
				info = val
				found = true
				break
			}
		}

		if found {
			foundCount++
			results = append(results, LookupResult{
				Target:    target,
				Found:     true,
				Name:      info.Name,
				IconLight: info.IconLight,
				IconDark:  info.IconDark,
			})
		} else {
			results = append(results, LookupResult{
				Target: target,
				Found:  false,
			})
		}
	}

	fmt.Println()
	fmt.Println(boxHeader(BLUE, "🔍", "AAGUID BATCH LOOKUP RESULTS"))

	for _, r := range results {
		if !r.Found {
			fmt.Printf("   🆔 ID: %s\n", r.Target)
			fmt.Printf("   └─ %s❌ Not Found in maps%s\n\n", RED, NC)
			continue
		}

		displayName := r.Name
		if displayName == "" || displayName == "null" {
			displayName = "Unknown Authenticator"
		}

		lightStatus := "❌"
		if len(r.IconLight) > 0 {
			lightStatus = "✅"
		}
		darkStatus := "❌"
		if len(r.IconDark) > 0 {
			darkStatus = "✅"
		}

		fmt.Printf("   🆔 ID: %s\n", r.Target)
		fmt.Printf("   ├─ [Metadata] %s%s%s\n", GREEN, displayName, NC)
		fmt.Printf("   └─ Assets: 🌞: %s | 🌒: %s\n\n", lightStatus, darkStatus)
	}
	fmt.Println(boxFooter(BLUE))

	fmt.Println(boxHeader(CYAN, "📊", "BATCH LOOKUP SUMMARY"))
	fmt.Printf("   %s🎯%s  %-15s Checked: %-3d | Found: %s%-3d%s\n", YELLOW, NC, "Results:", len(targets), GREEN, foundCount, NC)
	fmt.Println(boxFooter(CYAN))
	fmt.Println()
}
