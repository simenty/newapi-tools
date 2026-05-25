// NewAPI Tools - Internationalization (i18n) framework
package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

// Bundle holds translations for a specific language.
type Bundle struct {
	lang     string
	messages map[string]string
}

// NewBundle creates a new i18n Bundle by reading the embedded YAML file for the given language.
// If the specified language file is not found, it falls back to zh-CN.
func NewBundle(lang string) (*Bundle, error) {
	data, err := localeFS.ReadFile("locales/" + lang + ".yaml")
	if err != nil {
		// Fallback to default language zh-CN
		if lang != "zh-CN" {
			data, err = localeFS.ReadFile("locales/zh-CN.yaml")
			if err != nil {
				return nil, fmt.Errorf("i18n: failed to load locale file for %s (and fallback zh-CN): %w", lang, err)
			}
		} else {
			return nil, fmt.Errorf("i18n: failed to load locale file for %s: %w", lang, err)
		}
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("i18n: failed to parse locale file for %s: %w", lang, err)
	}

	messages := make(map[string]string)
	flattenYAML(raw, "", messages)

	return &Bundle{
		lang:     lang,
		messages: messages,
	}, nil
}

// T translates a key using the bundle's message map.
// If the key is not found, the key itself is returned.
// Supports fmt.Sprintf-style placeholders via args.
func (b *Bundle) T(key string, args ...any) string {
	msg, ok := b.messages[key]
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// Lang returns the language code of the bundle.
func (b *Bundle) Lang() string {
	return b.lang
}

// defaultBundle is the package-level singleton bundle used by the convenience functions.
var defaultBundle *Bundle

// Init initializes the default i18n bundle with the given language.
// If lang is empty, it is resolved from environment variables (LANG/LC_ALL) or defaults to zh-CN.
// This function should be called once at program startup.
func Init(lang string) error {
	if lang == "" {
		lang = resolveLangFromEnv()
	}
	bundle, err := NewBundle(lang)
	if err != nil {
		return err
	}
	defaultBundle = bundle
	return nil
}

// T translates a key using the default bundle.
// If the default bundle is nil (Init not called), the key itself is returned to prevent panics.
// Supports fmt.Sprintf-style placeholders via args.
func T(key string, args ...any) string {
	if defaultBundle == nil {
		return key
	}
	return defaultBundle.T(key, args...)
}

// resolveLangFromEnv determines the language from environment variables.
// Priority: LC_ALL > LANG > default zh-CN.
func resolveLangFromEnv() string {
	if lcAll := os.Getenv("LC_ALL"); lcAll != "" {
		return normalizeLang(lcAll)
	}
	if lang := os.Getenv("LANG"); lang != "" {
		return normalizeLang(lang)
	}
	return "zh-CN"
}

// normalizeLang converts locale strings like "zh_CN.UTF-8" or "en_US" to "zh-CN" or "en" format.
func normalizeLang(envVal string) string {
	// Remove encoding suffix (e.g., ".UTF-8")
	base := strings.Split(envVal, ".")[0]
	// Convert underscore to hyphen (e.g., zh_CN -> zh-CN)
	base = strings.ReplaceAll(base, "_", "-")

	// Map common locale values to our supported codes
	switch base {
	case "zh-CN", "zh-Hans-CN", "zh-Hans":
		return "zh-CN"
	case "zh-TW", "zh-Hant-CN", "zh-Hant", "zh-HK":
		return "zh-CN" // Traditional Chinese maps to Simplified for now
	case "en-US", "en-GB", "en-AU", "en-CA", "en-NZ", "en-IE":
		return "en"
	case "en":
		return "en"
	}

	// If it starts with "zh", default to zh-CN
	if strings.HasPrefix(base, "zh") {
		return "zh-CN"
	}
	// If it starts with "en", default to en
	if strings.HasPrefix(base, "en") {
		return "en"
	}

	// Default fallback
	return "zh-CN"
}

// flattenYAML recursively flattens a nested YAML map into dotted keys.
// For example: {"install": {"success": "安装成功！"}} becomes {"install.success": "安装成功！"}.
func flattenYAML(raw map[string]any, prefix string, result map[string]string) {
	for key, value := range raw {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := value.(type) {
		case string:
			result[fullKey] = v
		case map[string]any:
			flattenYAML(v, fullKey, result)
		case int:
			result[fullKey] = fmt.Sprintf("%d", v)
		case float64:
			result[fullKey] = fmt.Sprintf("%g", v)
		case bool:
			result[fullKey] = fmt.Sprintf("%t", v)
		case nil:
			result[fullKey] = ""
		default:
			result[fullKey] = fmt.Sprintf("%v", v)
		}
	}
}
