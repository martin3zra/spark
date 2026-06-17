package resources

import "embed"

//go:embed views/mail/*.html
var Views embed.FS
