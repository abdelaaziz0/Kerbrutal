package mutate

import (
    "fmt"
    "os"
    "strings"

    "github.com/abdelaaziz0/kerbrutal/util"
)

// MutationLevel controls how many permutations are generated.
type MutationLevel int

const (
    LevelStandard MutationLevel = iota // Tier 1 only (~8 per name)
    LevelExtended                       // Tier 1+2 (~15 per name)
    LevelFull                           // All tiers (~22 per name)
)

func GenerateUsernames(n Name, level MutationLevel) []string {
    var results []string

    f := n.First
    l := n.Last
    fi := n.FirstInitial
    li := n.LastInitial
    mi := n.MiddleInitial

    if l == "" {
        return nil
    }

    firstIsInitial := len(strings.TrimRight(f, ".")) <= 1

    if f != "" && !firstIsInitial {
        results = append(results,
            fi+"."+l,
            fi+l,
            f+"."+l,
            f+l,
            f+li,
            f+"."+li,
            l+"."+fi,
            l+fi,
        )
        for i := 1; i <= 4; i++ {
            results = append(results, fmt.Sprintf("%s%s%d", fi, l, i))
        }
    } else if fi != "" {
        results = append(results,
            fi+"."+l,
            fi+l,
            l+"."+fi,
            l+fi,
        )
    }

    if f != "" && !firstIsInitial {
        results = append(results, f)
    }
    results = append(results, l)

    if level < LevelExtended {
        return deduplicate(results)
    }

    if f != "" && !firstIsInitial {
        results = append(results,
            f+"_"+l,
            fi+"_"+l,
            l+"."+f,
            l+"_"+f,
            fi+"."+li,
        )
    }

    if level < LevelFull {
        return deduplicate(results)
    }

    if mi != "" && f != "" && !firstIsInitial {
        results = append(results,
            f+mi+l,
            fi+mi+l,
            f+"."+mi+"."+l,
            l+fi+mi,
        )
    }

    if strings.Contains(l, "-") {
        parts := strings.SplitN(l, "-", 2)
        if len(parts) == 2 && f != "" {
            collapsed := parts[0] + parts[1]
            results = append(results,
                fi+"."+collapsed,
                fi+collapsed,
                f+"."+collapsed,
                fi+"."+parts[0],
                fi+"."+parts[1],
            )
        }
    }

    return deduplicate(results)
}

func GenerateFromFile(path string, level MutationLevel, logger util.Logger) ([]string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading name file: %w", err)
    }

    content := string(data)
    content = strings.TrimPrefix(content, "\xef\xbb\xbf")

    lines := strings.Split(strings.TrimSpace(content), "\n")
    seen := make(map[string]bool)
    var all []string

    for lineNum, line := range lines {
        line = strings.TrimSpace(line)
        line = strings.TrimRight(line, "\r")

        if line == "" || strings.HasPrefix(line, "#") {
            continue 
        }

        name, err := ParseName(line)
        if err != nil {
            logger.Log.Warningf("Failed to parse name on line %d: %v", lineNum+1, err)
            continue
        }

        mutations := GenerateUsernames(name, level)
        for _, u := range mutations {
            if !seen[u] {
                seen[u] = true
                all = append(all, u)
            }
        }
    }

    return all, nil
}

func deduplicate(ss []string) []string {
    seen := make(map[string]bool, len(ss))
    result := make([]string, 0, len(ss))
    for _, s := range ss {
        if s != "" && !seen[s] {
            seen[s] = true
            result = append(result, s)
        }
    }
    return result
}
