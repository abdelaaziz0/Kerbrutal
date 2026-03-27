package mutate

import (
    "fmt"
    "strings"
    "unicode"

    "golang.org/x/text/runes"
    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"
)

// Name represents a parsed person name.
type Name struct {
    First         string // full first name, lowercase, ascii-normalized
    Middle        string // full middle name (if any)
    Last          string // full last name, lowercase, ascii-normalized
    FirstInitial  string // first letter of first name
    MiddleInitial string // first letter of middle name
    LastInitial   string // first letter of last name
    Original      string // original input line
}

// Known prefixes and suffixes to strip
var prefixes = []string{
    "dr.", "dr", "mr.", "mr", "mrs.", "mrs", "ms.", "ms",
    "prof.", "prof", "sir", "rev.", "rev",
}
var suffixes = []string{
    "jr.", "jr", "sr.", "sr", "ii", "iii", "iv", "v",
    "phd", "ph.d", "md", "m.d", "esq", "esq.", "cpa", "dds",
}

// ParseName parses a raw name string into a structured Name.
func ParseName(raw string) (Name, error) {
    raw = strings.TrimSpace(raw)
    if raw == "" {
        return Name{}, fmt.Errorf("empty name")
    }

    n := Name{Original: raw}

    // Lowercase everything
    line := strings.ToLower(raw)

    // Strip prefixes
    for _, p := range prefixes {
        if strings.HasPrefix(line, p+" ") {
            line = strings.TrimSpace(line[len(p):])
            break
        }
    }

    // Strip suffixes
    for _, s := range suffixes {
        if strings.HasSuffix(line, " "+s) {
            line = strings.TrimSpace(line[:len(line)-len(s)-1])
            break
        }
        // Also handle comma-separated suffix: "Doe, John, III"
        if strings.HasSuffix(line, ", "+s) {
            line = strings.TrimSpace(line[:len(line)-len(s)-2])
            break
        }
    }

    // Detect "Last, First" format (comma-separated)
    if idx := strings.Index(line, ","); idx > 0 {
        lastName := strings.TrimSpace(line[:idx])
        remainder := strings.TrimSpace(line[idx+1:])
        parts := strings.Fields(remainder)
        if len(parts) >= 1 {
            n.Last = normalizeStr(lastName)
            n.First = normalizeStr(parts[0])
            if len(parts) >= 2 {
                n.Middle = normalizeStr(parts[1])
            }
            fillInitials(&n)
            return n, nil
        }
    }

    // Standard "First [Middle] Last" format
    parts := strings.Fields(line)
    switch len(parts) {
    case 1:
        // Could be just a username already — treat as last name
        n.Last = normalizeStr(parts[0])
    case 2:
        n.First = normalizeStr(parts[0])
        n.Last = normalizeStr(parts[1])
    case 3:
        n.First = normalizeStr(parts[0])
        n.Middle = normalizeStr(parts[1])
        n.Last = normalizeStr(parts[2])
    default:
        // 4+ parts: first = parts[0], last = last part,
        // middle = everything in between
        n.First = normalizeStr(parts[0])
        n.Last = normalizeStr(parts[len(parts)-1])
        midParts := make([]string, 0, len(parts)-2)
        for _, p := range parts[1 : len(parts)-1] {
            midParts = append(midParts, normalizeStr(p))
        }
        n.Middle = strings.Join(midParts, " ")
    }

    fillInitials(&n)
    return n, nil
}

func fillInitials(n *Name) {
    if n.First != "" {
        clean := strings.TrimRight(n.First, ".")
        if len(clean) > 0 {
            n.FirstInitial = string([]rune(clean)[0])
        }
    }
    if n.Middle != "" {
        clean := strings.TrimRight(n.Middle, ".")
        if len(clean) > 0 {
            n.MiddleInitial = string([]rune(clean)[0])
        }
    }
    if n.Last != "" {
        n.LastInitial = string([]rune(n.Last)[0])
    }
}

// normalizeStr lowercases, strips accents/diacritics, removes
// apostrophes, and collapses whitespace.
func normalizeStr(s string) string {
    s = strings.ToLower(s)

    // Remove diacritics: NFD decompose, then strip combining marks
    t := transform.Chain(
        norm.NFD,
        runes.Remove(runes.In(unicode.Mn)), // Mn = Mark, Nonspacing
        norm.NFC,
    )
    result, _, _ := transform.String(t, s)

    // Remove apostrophes and dots
    result = strings.ReplaceAll(result, "'", "")
    result = strings.ReplaceAll(result, "\u2019", "") // right single quote
    result = strings.ReplaceAll(result, ".", "")

    return strings.TrimSpace(result)
}
