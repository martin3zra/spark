package app

import "github.com/martin3zra/forge/i18n"

func trans(namespaces ...string) map[string]string {
	translations, err := i18n.LoadTranslations(localesFS, defaultLang, fallbackLang, namespaces...)
	if err != nil {
		panic(err)
	}
	return translations
}
