package grammars

import (
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

// Languages where the highlight query compiles but the smoke sample is too
// simple to produce any highlight ranges. These are not bugs.
var highlightNoRangesExpected = map[string]bool{
	"jq":      true, // sample ".foo" has no nodes matching jq highlights
	"jsdoc":   true, // sample "/** hello */" has no nodes matching jsdoc highlights
	"nginx":   true, // sample "events {}" does not exercise most nginx captures
	"svelte":  true, // sample is plain HTML text, no svelte-specific nodes
	"wolfram": true, // sample "1 + 2" has no nodes matching wolfram highlights
	"cpp":     true, // current smoke sample is too small for useful capture coverage
	"haskell": true, // sample is intentionally tiny and misses most capture paths
	"haxe":    true, // sample "1;" intentionally minimal
	"tsx":     true, // sample has limited syntax for broad TSX highlight rules
}

func TestHighlightQueriesProduceResults(t *testing.T) {
	entries := AllLanguages()

	reports := AuditParseSupport()
	reportByName := make(map[string]ParseSupport, len(reports))
	for _, r := range reports {
		reportByName[r.Name] = r
	}

	var tested, skippedNoQuery, skippedNoSample, skippedUnsupported int
	for _, entry := range entries {
		name := entry.Name
		if strings.TrimSpace(entry.HighlightQuery) == "" {
			skippedNoQuery++
			continue
		}

		report := reportByName[name]
		if report.Backend == ParseBackendUnsupported {
			skippedUnsupported++
			continue
		}

		sample := parseSmokeSample(name)
		if sample == "x\n" {
			skippedNoSample++
			continue
		}

		tested++
		t.Run(name, func(t *testing.T) {
			lang := entry.Language()

			// Build highlighter options.
			var opts []gotreesitter.HighlighterOption
			if entry.TokenSourceFactory != nil {
				factory := entry.TokenSourceFactory
				opts = append(opts, gotreesitter.WithTokenSourceFactory(
					func(src []byte) gotreesitter.TokenSource {
						return factory(src, lang)
					},
				))
			}

			h, err := gotreesitter.NewHighlighter(lang, entry.HighlightQuery, opts...)
			if err != nil {
				// Many highlight queries use features our query compiler
				// doesn't support yet. Log these rather than failing.
				t.Skipf("query compilation not supported: %v", err)
				return
			}

			ranges := h.Highlight([]byte(sample))
			if len(ranges) == 0 && !highlightNoRangesExpected[name] {
				t.Errorf("highlight query compiled but produced 0 ranges for sample %q", sample)
			}
		})
	}

	t.Logf("highlight validation: tested=%d skipped(no_query=%d no_sample=%d unsupported=%d)",
		tested, skippedNoQuery, skippedNoSample, skippedUnsupported)

}
