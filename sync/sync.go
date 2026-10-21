package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// --- Configuration ---
const (
	defaultSourceFile = "../aaguid.go"
	defaultTargetFile = "../json/aaguids.json"
	boxInnerWidth     = 63
)

// --- UI Colors ---
const (
	GREEN   = "\033[0;32m"
	RED     = "\033[0;31m"
	YELLOW  = "\033[0;33m"
	BLUE    = "\033[0;34m"
	CYAN    = "\033[0;36m"
	MAGENTA = "\033[0;35m"
	BOLD    = "\033[1m"
	DIM     = "\033[2m"
	NC      = "\033[0m"
)

type Entry struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	IconLight string `json:"icon_light"`
	IconDark  string `json:"icon_dark"`
	Category  string `json:"category"`
	OldName   string `json:"-"`
}

func boxHeader(color, icon, title string) string {
	titleRunes := utf8.RuneCountInString(title)
	dashes := max(boxInnerWidth-(2+1+2+1+titleRunes+1), 1)
	return fmt.Sprintf("%s┌── %s %s %s┐%s", color, icon, title, strings.Repeat("─", dashes), NC)
}

func boxFooter(color string) string {
	return fmt.Sprintf("%s└%s┘%s", color, strings.Repeat("─", boxInnerWidth), NC)
}

func formatEntry(entries []Entry) string {
	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		displayName := e.Name
		if e.OldName != "" && e.OldName != e.Name {
			displayName = fmt.Sprintf("%s → %s", e.OldName, e.Name)
		} else if e.Name == "" || e.Name == "null" {
			displayName = "Unknown Authenticator"
		}

		lightStatus := "❌"
		if len(e.IconLight) > 0 {
			lightStatus = "✅"
		}
		darkStatus := "❌"
		if len(e.IconDark) > 0 {
			darkStatus = "✅"
		}

		sb.WriteString(fmt.Sprintf(
			"\n      🆔 ID: %s\n      ├─ [%s] %s\n      └─ Assets: 🌞: %s | 🌒: %s",
			e.UUID, e.Category, displayName, lightStatus, darkStatus,
		))
	}
	return sb.String()
}

func showHelp(progName string) {
	fmt.Println()
	fmt.Println(boxHeader(MAGENTA, "🔧", "AAGUID SYNC - HELP MENU"))
	fmt.Printf("  %s📋  %-15s%s %s [options]\n\n", CYAN, "USAGE:", NC, progName)
	fmt.Printf("  %s⚡  %-15s%s\n", CYAN, "OPTIONS:", NC)
	fmt.Printf("      %s%-18s%s Search for a specific authenticator\n", YELLOW, "-f, --find <str>", NC)
	fmt.Printf("      %s%-18s%s Preview changes without saving\n", YELLOW, "-d, --dry-run", NC)
	fmt.Printf("      %s%-18s%s Show this beautiful menu\n\n", YELLOW, "-h, --help", NC)
	fmt.Printf("  %s📖  %-15s%s\n", CYAN, "EXAMPLES:", NC)
	fmt.Printf("      %s --find \"Yubico\"\n", progName)
	fmt.Printf("      %s --dry-run\n", progName)
	fmt.Println(boxFooter(MAGENTA))
	fmt.Println()
	os.Exit(0)
}

func humanFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"K", "M", "G", "T", "P", "E"}
	val := float64(bytes) / float64(div)
	if val >= 10 {
		return fmt.Sprintf("%.0f%s", val, units[exp])
	}
	return fmt.Sprintf("%.1f%s", val, units[exp])
}

func extractCategory(sourceContent, varName, category string) []Entry {
	blockRe := regexp.MustCompile(`(?s)var ` + regexp.QuoteMeta(varName) + `.*?\n\}`)
	lineRe := regexp.MustCompile(`(?m)^\t*MustParseAAGUID\("([^"]+)"\):\s*\{\s*Name:\s*"([^"]+)"(?:,\s*IconLight:\s*"([^"]+)")?(?:,\s*IconDark:\s*"([^"]+)")?.*`)

	var results []Entry
	blocks := blockRe.FindAllString(sourceContent, -1)
	for _, block := range blocks {
		matches := lineRe.FindAllStringSubmatch(block, -1)
		for _, m := range matches {
			results = append(results, Entry{
				UUID:      m[1],
				Name:      m[2],
				IconLight: m[3],
				IconDark:  m[4],
				Category:  category,
			})
		}
	}
	return results
}

func groupAndDeduplicate(raw []Entry) (extracted []Entry, duplicatesFlat []Entry, duplicateGroupCount int) {
	sorted := make([]Entry, len(raw))
	copy(sorted, raw)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].UUID < sorted[j].UUID
	})

	var i int
	for i < len(sorted) {
		j := i + 1
		for j < len(sorted) && sorted[j].UUID == sorted[i].UUID {
			j++
		}
		extracted = append(extracted, sorted[i])

		if j-i > 1 {
			duplicateGroupCount++
			duplicatesFlat = append(duplicatesFlat, sorted[i:j]...)
		}
		i = j
	}
	return extracted, duplicatesFlat, duplicateGroupCount
}

func writePrettyJSON(filename string, entries []Entry) error {
	if entries == nil {
		entries = []Entry{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(entries); err != nil {
		return err
	}
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

func main() {
	startTime := time.Now()
	sourceFile := defaultSourceFile
	targetFile := defaultTargetFile
	dryRun := false
	searchQuery := ""

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-d" || arg == "--dry-run":
			dryRun = true
		case arg == "-f" || arg == "--find":
			if i+1 < len(args) {
				searchQuery = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--find="):
			searchQuery = strings.TrimPrefix(arg, "--find=")
		case strings.HasPrefix(arg, "-f="):
			searchQuery = strings.TrimPrefix(arg, "-f=")
		case arg == "-h" || arg == "--help":
			showHelp(filepath.Base(os.Args[0]))
		}
	}

	wasReset := false
	var existingEntries []Entry

	targetBytes, err := os.ReadFile(targetFile)
	if err != nil || !json.Valid(targetBytes) || json.Unmarshal(targetBytes, &existingEntries) != nil {
		_ = os.WriteFile(targetFile, []byte("[]\n"), 0644)
		existingEntries = []Entry{}
		wasReset = true
	}

	var oldSizeBytes int64
	if info, err := os.Stat(targetFile); err == nil {
		oldSizeBytes = info.Size()
	}
	oldSizeHuman := humanFileSize(oldSizeBytes)

	var rawExtracted []Entry
	if srcBytes, err := os.ReadFile(sourceFile); err == nil {
		srcContent := string(srcBytes)
		rawExtracted = append(rawExtracted, extractCategory(srcContent, "PasskeyAuthenticatorAAGUIDs", "Passkey")...)
		rawExtracted = append(rawExtracted, extractCategory(srcContent, "MetadataAAGUIDs", "Metadata")...)
	}

	extractedEntries, duplicatesJSON, duplicatesVal := groupAndDeduplicate(rawExtracted)

	// --- DIFF  ---
	existingMap := make(map[string]Entry, len(existingEntries))
	for _, e := range existingEntries {
		existingMap[e.UUID] = e
	}

	extractedMap := make(map[string]Entry, len(extractedEntries))
	for _, e := range extractedEntries {
		extractedMap[e.UUID] = e
	}

	var newEntries []Entry
	var nameChanges []Entry
	for _, n := range extractedEntries {
		o, exists := existingMap[n.UUID]
		if !exists {
			newEntries = append(newEntries, n)
		} else if o.Name != n.Name || o.IconLight != n.IconLight || o.IconDark != n.IconDark {
			changed := n
			changed.OldName = o.Name
			nameChanges = append(nameChanges, changed)
		}
	}

	var deletedEntries []Entry
	for _, o := range existingEntries {
		if _, exists := extractedMap[o.UUID]; !exists {
			deletedEntries = append(deletedEntries, o)
		}
	}

	newCount := len(newEntries)
	delCount := len(deletedEntries)
	updCount := len(nameChanges)

	fmt.Println()
	fmt.Println(boxHeader(BLUE, "🔐", "AAGUID SYNC HELPER"))
	fmt.Printf("  📂  %-15s %s%s%s\n", "Reading from:", DIM, sourceFile, NC)

	if wasReset {
		fmt.Printf("  🧹  %-15s %sTarget file was reset.%s\n", "Note:", YELLOW, NC)
	}

	if dryRun {
		fmt.Printf("  ✨  %-15s %sChanges will not be saved.%s\n", "Preview Mode:", YELLOW, NC)
	}

	fmt.Println(boxFooter(BLUE))

	fmt.Println(boxHeader(YELLOW, "💡", "TIP"))
	fmt.Printf("  🔍  %-15s %sUse -f <name> to search authenticators.%s\n", "Search:", YELLOW, NC)
	fmt.Println(boxFooter(YELLOW))

	if searchQuery != "" {
		totalItems := len(extractedEntries)
		var results []Entry
		searchRe, reErr := regexp.Compile("(?i)" + searchQuery)
		for _, e := range extractedEntries {
			if reErr == nil {
				if searchRe.MatchString(e.Name) {
					results = append(results, e)
				}
			} else if strings.Contains(strings.ToLower(e.Name), strings.ToLower(searchQuery)) {
				results = append(results, e)
			}
		}

		fmt.Println()
		fmt.Println(boxHeader(CYAN, "🔍", fmt.Sprintf("SEARCH RESULTS: \"%s\"", searchQuery)))
		fmt.Printf("  📊  %-15s %sItems: %d | DB Size: (%s)%s\n", "Database:", DIM, totalItems, oldSizeHuman, NC)

		if len(results) == 0 {
			fmt.Printf("  %s❌  No authenticators found matching \"%s\"%s\n", RED, searchQuery, NC)
		} else {
			formatted := formatEntry(results)
			for _, line := range strings.Split(formatted, "\n") {
				fmt.Printf("  %s\n", line)
			}
		}

		fmt.Println()
		fmt.Println(boxFooter(CYAN))
		os.Exit(0)
	}

	fmt.Println()
	fmt.Println(boxHeader(BLUE, "📝", "CHANGES TO BE MADE"))
	if newCount+delCount+updCount+duplicatesVal == 0 {
		fmt.Printf("  ✨  %sEverything looks perfect! No changes needed.%s\n", DIM, NC)
	} else {
		if newCount > 0 {
			fmt.Printf("  %s%s➕  ADDED (%d)%s\n", GREEN, BOLD, newCount, NC)
			fmt.Println(formatEntry(newEntries))
		}
		if updCount > 0 {
			fmt.Printf("\n  %s%s📝  UPDATED (%d)%s\n", YELLOW, BOLD, updCount, NC)
			fmt.Println(formatEntry(nameChanges))
		}
		if delCount > 0 {
			fmt.Printf("\n  %s%s➖  REMOVED (%d)%s\n", RED, BOLD, delCount, NC)
			fmt.Println(formatEntry(deletedEntries))
		}
		if duplicatesVal > 0 {
			fmt.Printf("\n  %s%s👯  DUPLICATES IN SOURCE (%d)%s\n", MAGENTA, BOLD, duplicatesVal, NC)
			fmt.Println(formatEntry(duplicatesJSON))
		}
	}
	fmt.Println(boxFooter(BLUE))

	if !dryRun {
		sortedByName := make([]Entry, len(extractedEntries))
		copy(sortedByName, extractedEntries)
		sort.SliceStable(sortedByName, func(i, j int) bool {
			return sortedByName[i].Name < sortedByName[j].Name
		})
		_ = writePrettyJSON(targetFile, sortedByName)
	}

	var newSizeBytes int64
	if info, err := os.Stat(targetFile); err == nil {
		newSizeBytes = info.Size()
	}
	newSizeHuman := humanFileSize(newSizeBytes)

	var finalTargetEntries []Entry
	if finalBytes, err := os.ReadFile(targetFile); err == nil {
		_ = json.Unmarshal(finalBytes, &finalTargetEntries)
	}
	totalCount := len(finalTargetEntries)

	metadataCount := 0
	passkeyCount := 0
	lightIcons := 0
	darkIcons := 0
	for _, e := range extractedEntries {
		switch e.Category {
		case "Metadata":
			metadataCount++
		case "Passkey":
			passkeyCount++
		}
		if len(e.IconLight) > 0 {
			lightIcons++
		}
		if len(e.IconDark) > 0 {
			darkIcons++
		}
	}

	durationMs := time.Since(startTime).Milliseconds()

	fmt.Println()
	fmt.Println(boxHeader(CYAN, "📊", "FILE SUMMARY"))
	fmt.Printf("  %s📂%s  %-15s Metadata: %-5d |  Passkey: %-5d\n", YELLOW, NC, "Types:", metadataCount, passkeyCount)
	fmt.Printf("  %s🎨%s  %-15s 🌞: %-11d |  🌒: %-5d\n", GREEN, NC, "Icons:", lightIcons, darkIcons)
	fmt.Printf("  %s📄%s  %-15s %s%s%s → %s%s bytes%s\n", BLUE, NC, "File Size:", DIM, oldSizeHuman, NC, BOLD, newSizeHuman, NC)
	fmt.Printf("  %s🔢%s  %-15s %s%s%d authenticators%s\n", CYAN, NC, "Total Items:", CYAN, BOLD, totalCount, NC)
	fmt.Printf("  %s👯%s  %-15s %s%s%d duplicates found%s\n", RED, NC, "Duplicates:", RED, BOLD, duplicatesVal, NC)
	fmt.Printf("  %s✨%s  %-15s %s+%d%s | %s-%d%s | %s~%d%s\n", YELLOW, NC, "Quick Delta:", GREEN, newCount, NC, RED, delCount, NC, YELLOW, updCount, NC)
	fmt.Printf("  %s⏳%s  %-15s %dms\n", NC, NC, "Process Time:", durationMs)
	fmt.Println(boxFooter(CYAN))
	fmt.Println()
}
