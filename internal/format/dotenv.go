package format

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ParseDotEnv reads a .env file and returns key-value pairs.
func ParseDotEnv(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := line[idx+1:]

		value = parseValue(value)
		result[key] = value
	}

	return result, scanner.Err()
}

func parseValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}

	// Double-quoted value
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		inner := raw[1 : len(raw)-1]
		inner = strings.ReplaceAll(inner, `\\`, "\x00")
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\n`, "\n")
		inner = strings.ReplaceAll(inner, `\t`, "\t")
		inner = strings.ReplaceAll(inner, "\x00", `\`)
		return inner
	}

	// Single-quoted value (literal)
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1]
	}

	// Unquoted: strip inline comments
	if idx := strings.IndexByte(raw, '#'); idx > 0 {
		raw = strings.TrimSpace(raw[:idx])
	}

	return raw
}

// WriteDotEnv writes a .env file with sorted keys and a header comment.
func WriteDotEnv(w io.Writer, secrets map[string]string, env string, version int, updatedAt string) error {
	fmt.Fprintf(w, "# envs3 | Environment: %s | Version: %d | Updated: %s\n\n", env, version, updatedAt)

	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := secrets[key]
		if needsQuoting(value) {
			value = quoteValue(value)
		}
		fmt.Fprintf(w, "%s=%s\n", key, value)
	}

	return nil
}

func needsQuoting(v string) bool {
	if v == "" {
		return false
	}
	if strings.ContainsAny(v, " \t#=\"\n\\") {
		return true
	}
	if v[0] == ' ' || v[len(v)-1] == ' ' {
		return true
	}
	return false
}

func quoteValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	v = strings.ReplaceAll(v, "\t", `\t`)
	return `"` + v + `"`
}

// CanonicalJSON produces deterministic JSON bytes for the secrets map
// (sorted keys, no whitespace) suitable for checksum computation.
func CanonicalJSON(secrets map[string]EncryptedSecret) ([]byte, error) {
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]canonicalEntry, 0, len(keys))
	for _, k := range keys {
		ordered = append(ordered, canonicalEntry{Key: k, Value: secrets[k]})
	}

	// Build a JSON object with sorted keys
	buf := []byte("{")
	for i, entry := range ordered {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyJSON, err := json.Marshal(entry.Key)
		if err != nil {
			return nil, err
		}
		valJSON, err := json.Marshal(entry.Value)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyJSON...)
		buf = append(buf, ':')
		buf = append(buf, valJSON...)
	}
	buf = append(buf, '}')

	return buf, nil
}

type canonicalEntry struct {
	Key   string
	Value EncryptedSecret
}
