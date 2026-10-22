package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// --- Configuration ---
const (
	targetDir         = "../"
	defaultTargetFile = "aaguid.go"
	boxInnerWidth     = 63
	passkeyRepoURL    = "https://raw.githubusercontent.com/passkeydeveloper/passkey-authenticator-aaguids/main/aaguid.json"
	fidoMDSURL        = "https://mds3.fidoalliance.org/"
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

type AAGUIDInfo struct {
	Name      string `json:"name"`
	IconLight string `json:"icon_light"`
	IconDark  string `json:"icon_dark"`
}

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
			"\n     🆔 ID: %s\n     ├─ [%s] %s\n     └─ Assets: 🌞: %s | 🌒: %s",
			e.UUID, e.Category, displayName, lightStatus, darkStatus,
		))
	}
	return sb.String()
}

func showHelp(progName string) {
	fmt.Println()
	fmt.Println(boxHeader(MAGENTA, "🔧", "AAGUID GENERATOR - HELP MENU"))
	fmt.Printf("  %s📋  %-15s%s %s [options]\n\n", CYAN, "USAGE:", NC, progName)
	fmt.Printf("  %s⚡  %-15s%s\n", CYAN, "OPTIONS:", NC)
	fmt.Printf("      %s%-18s%s Search for a specific authenticator\n", YELLOW, "-f, --find <str>", NC)
	fmt.Printf("      %s%-18s%s Preview changes without saving\n", YELLOW, "-d, --dry-run", NC)
	fmt.Printf("      %s%-18s%s Show this help menu\n\n", YELLOW, "-h, --help", NC)
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

func formatUUID(raw string) string {
	cleaned := strings.NewReplacer("-", "", "{", "", "}", "").Replace(raw)
	if len(cleaned) != 32 {
		return raw
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", cleaned[0:8], cleaned[8:12], cleaned[12:16], cleaned[16:20], cleaned[20:32])
}

func loadExistingEntries(targetFile string) []Entry {
	var results []Entry
	content, err := os.ReadFile(targetFile)
	if err != nil {
		return results
	}

	reEntry := regexp.MustCompile(`MustParseAAGUID\("(.*?)"\)\s*:\s*\{\s*Name:\s*"(.*?)",\s*IconLight:\s*"(.*?)",\s*IconDark:\s*"(.*?)",\s*\},`)
	matches := reEntry.FindAllStringSubmatch(string(content), -1)

	for _, m := range matches {
		if len(m) == 5 {
			results = append(results, Entry{
				UUID:      m[1],
				Name:      m[2],
				IconLight: m[3],
				IconDark:  m[4],
				Category:  "Metadata",
			})
		}
	}
	return results
}

func fetchPasskeyRepo() (map[string]AAGUIDInfo, error) {
	resp, err := http.Get(passkeyRepoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s (status %d)", resp.Status, resp.StatusCode)
	}

	var raw map[string]AAGUIDInfo
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	result := make(map[string]AAGUIDInfo)
	for uuid, info := range raw {
		result[formatUUID(uuid)] = info
	}
	return result, nil
}

func fetchFidoMDS() (map[string]AAGUIDInfo, error) {
	resp, err := http.Get(fidoMDSURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s (status %d)", resp.Status, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyStr := string(bodyBytes)
	if strings.HasPrefix(strings.TrimSpace(bodyStr), "<") {
		return nil, fmt.Errorf("received HTML response instead of JWT blob")
	}

	parts := strings.Split(bodyStr, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid JWT structure from FIDO MDS")
	}

	decodedBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		decodedBytes, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}
	}

	var mdsData struct {
		Entries []struct {
			AAGUID            *string `json:"aaguid"`
			MetadataStatement struct {
				Description string  `json:"description"`
				Icon        *string `json:"icon"`
			} `json:"metadataStatement"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(decodedBytes, &mdsData); err != nil {
		return nil, err
	}

	result := make(map[string]AAGUIDInfo)
	for _, entry := range mdsData.Entries {
		if entry.AAGUID != nil && *entry.AAGUID != "" {
			uuidStr := formatUUID(*entry.AAGUID)
			icon := ""
			if entry.MetadataStatement.Icon != nil {
				icon = *entry.MetadataStatement.Icon
			}
			result[uuidStr] = AAGUIDInfo{
				Name:      entry.MetadataStatement.Description,
				IconLight: icon,
				IconDark:  icon,
			}
		}
	}
	return result, nil
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

func main() {
	startTime := time.Now()

	targetFile := filepath.Join(targetDir, defaultTargetFile)

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

	var oldSizeBytes int64
	if info, err := os.Stat(targetFile); err == nil {
		oldSizeBytes = info.Size()
	}
	oldSizeHuman := humanFileSize(oldSizeBytes)

	fmt.Println()
	fmt.Println(boxHeader(BLUE, "🔐", "AAGUID CONCURRENT GENERATOR & SYNC"))

	if dryRun {
		fmt.Printf("  ✨  %-15s %sChanges will not be saved.%s\n", "Preview Mode:", YELLOW, NC)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil && !dryRun {
		panic(fmt.Sprintf("failed to create target directory: %v", err))
	}

	if _, err := os.Stat(targetFile); os.IsNotExist(err) && !dryRun {
		initialSkeleton := `// Code generated by gen/gen.go. DO NOT EDIT.
package aaguids

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type AAGUID [16]byte

type AAGUIDInfo struct {
	Name      string
	IconDark  string
	IconLight string
}

func MustParseAAGUID(uuidStr string) AAGUID {
	cleaned := strings.ReplaceAll(uuidStr, "-", "")

	var aaguid AAGUID
	data, err := hex.DecodeString(cleaned)

	if err != nil || len(data) != 16 {
		panic(fmt.Sprintf("invalid AAGUID: %s", uuidStr))
	}

	copy(aaguid[:], data)

	return aaguid
}

var MetadataAAGUIDs = map[AAGUID]AAGUIDInfo{
}
`
		_ = os.WriteFile(targetFile, []byte(initialSkeleton), 0644)
	}

	existingEntries := loadExistingEntries(targetFile)
	fmt.Printf("  🔒  %-15s Loaded %d baseline entries from %s\n", "Baseline:", len(existingEntries), targetFile)

	var rawExtracted []Entry
	rawExtracted = append(rawExtracted, existingEntries...)

	type fetchResult struct {
		source string
		count  int
		err    error
	}

	resultsChan := make(chan fetchResult, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		data, err := fetchPasskeyRepo()
		count := 0
		if err == nil {
			count = len(data)
			for uuid, info := range data {
				rawExtracted = append(rawExtracted, Entry{
					UUID:      uuid,
					Name:      info.Name,
					IconLight: info.IconLight,
					IconDark:  info.IconDark,
					Category:  "Passkey",
				})
			}
		}
		resultsChan <- fetchResult{source: "Passkey Repo", count: count, err: err}
	}()

	go func() {
		defer wg.Done()
		data, err := fetchFidoMDS()
		count := 0
		if err == nil {
			count = len(data)
			for uuid, info := range data {
				rawExtracted = append(rawExtracted, Entry{
					UUID:      uuid,
					Name:      info.Name,
					IconLight: info.IconLight,
					IconDark:  info.IconDark,
					Category:  "Metadata",
				})
			}
		}
		resultsChan <- fetchResult{source: "FIDO MDS", count: count, err: err}
	}()

	wg.Wait()
	close(resultsChan)

	// --- SOURCE HEALTH CHECK BOX ---
	fmt.Println()
	fmt.Println(boxHeader(CYAN, "🌐", "SOURCE HEALTH CHECK"))
	for i := 0; i < 2; i++ {
		res := <-resultsChan
		if res.err != nil {
			fmt.Printf("  %s❌  %-15s Status: FAILED (%v)%s\n", RED, res.source, res.err, NC)
		} else {
			fmt.Printf("  %s✅  %-15s Status: OK (Fetched %d items)%s\n", GREEN, res.source, res.count, NC)
		}
	}
	fmt.Println(boxFooter(CYAN))

	extractedEntries, duplicatesJSON, duplicatesVal := groupAndDeduplicate(rawExtracted)

	// --- DIFF ---
	existingMap := make(map[string]Entry, len(existingEntries))
	for _, e := range existingEntries {
		existingMap[e.UUID] = e
	}

	var newEntries []Entry
	for _, n := range extractedEntries {
		if _, exists := existingMap[n.UUID]; !exists {
			newEntries = append(newEntries, n)
		}
	}

	newCount := len(newEntries)
	delCount := 0
	updCount := 0

	if searchQuery != "" {
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
		if len(results) == 0 {
			fmt.Printf("  %s❌  No authenticators found matching \"%s\"%s\n", RED, searchQuery, NC)
		} else {
			formatted := formatEntry(results)
			for _, line := range strings.Split(formatted, "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
		fmt.Println(boxFooter(CYAN))
		os.Exit(0)
	}

	fmt.Println()
	fmt.Println(boxHeader(BLUE, "📝", "CHANGES TO BE MADE"))
	if newCount+duplicatesVal == 0 {
		fmt.Printf("  ✨  %sEverything looks pristine! No new items found.%s\n", DIM, NC)
	} else {
		if newCount > 0 {
			fmt.Printf("  %s%s➕  ADDED (%d)%s\n", GREEN, BOLD, newCount, NC)
			fmt.Println(formatEntry(newEntries))
		}
		if duplicatesVal > 0 {
			fmt.Printf("\n  %s%s👯  DUPLICATES CAUGHT & REMOVED (%d)%s\n", MAGENTA, BOLD, duplicatesVal, NC)
			fmt.Println(formatEntry(duplicatesJSON))
		}
	}
	fmt.Println(boxFooter(BLUE))

	if !dryRun && len(newEntries) > 0 {
		contentBytes, err := os.ReadFile(targetFile)
		if err != nil {
			panic(fmt.Sprintf("failed to read target file: %v", err))
		}
		contentStr := string(contentBytes)

		var sb strings.Builder
		for _, e := range newEntries {
			fmt.Fprintf(&sb, "\tMustParseAAGUID(%q): {\n\t\tName: %q,\n\t\tIconLight: %q,\n\t\tIconDark: %q,\n\t},\n",
				e.UUID, e.Name, e.IconLight, e.IconDark)
		}

		lastIdx := strings.LastIndex(contentStr, "}")
		if lastIdx == -1 {
			panic("invalid aaguid.go structure: closing brace not found")
		}

		updatedContent := contentStr[:lastIdx] + sb.String() + contentStr[lastIdx:]

		formattedCode, err := format.Source([]byte(updatedContent))
		if err != nil {
			_ = os.WriteFile(targetFile, []byte(updatedContent), 0644)
			panic(fmt.Sprintf("failed to format generated code: %v", err))
		}

		_ = os.WriteFile(targetFile, formattedCode, 0644)
	}

	var newSizeBytes int64
	if info, err := os.Stat(targetFile); err == nil {
		newSizeBytes = info.Size()
	}
	newSizeHuman := humanFileSize(newSizeBytes)

	totalCount := len(extractedEntries)

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
	fmt.Printf("  %s👯%s  %-15s %s%s%d duplicates removed%s\n", RED, NC, "Duplicates:", RED, BOLD, duplicatesVal, NC)
	fmt.Printf("  %s✨%s  %-15s %s+%d%s | %s-%d%s | %s~%d%s\n", YELLOW, NC, "Quick Delta:", GREEN, newCount, NC, RED, delCount, NC, YELLOW, updCount, NC)
	fmt.Printf("  %s⏳%s  %-15s %dms (Parallel Fetch)\n", NC, NC, "Process Time:", durationMs)
	fmt.Println(boxFooter(CYAN))
	fmt.Println()
}
