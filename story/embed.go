// Package story embeds the Night Express story graph and its texts.
package story

import "embed"

// FS holds graph.json and text/{uz,ru,en}.json.
//
//go:embed graph.json text/*.json
var FS embed.FS
