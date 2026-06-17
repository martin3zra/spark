package app

import "embed"

//go:embed locales/*.json
var localesFS embed.FS

const (
	defaultLang  = "en"
	fallbackLang = "en"
)
