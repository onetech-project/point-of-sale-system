package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Translations map[string]map[string]string

var translations = make(map[string]Translations)

func LoadTranslations(localesDir string) error {
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		locale := strings.TrimSuffix(entry.Name(), ".json")
		filePath := filepath.Join(localesDir, entry.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read translation file %s: %w", filePath, err)
		}

		var trans Translations
		if err := json.Unmarshal(data, &trans); err != nil {
			return fmt.Errorf("failed to parse translation file %s: %w", filePath, err)
		}

		translations[locale] = trans
	}

	return nil
}

func Translate(locale, key string) string {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return key
	}

	trans, ok := translations[locale]
	if !ok {
		trans = translations["en"]
	}

	section, ok := trans[parts[0]]
	if !ok {
		return key
	}

	message, ok := section[parts[1]]
	if !ok {
		return key
	}

	return message
}

func GetSupportedLocales() []string {
	locales := make([]string, 0, len(translations))
	for locale := range translations {
		locales = append(locales, locale)
	}
	return locales
}
