package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// The website's CLI reference is written by hand so flags can be grouped and explained, which
// means nothing regenerates it when a flag changes. This keeps the page and the CLI in step:
// every flag must be documented, and every documented flag must exist.
func TestCLIReferenceCoversEveryFlag(t *testing.T) {
	page := filepath.Join("..", "..", "website", "src", "content", "docs", "reference", "cli.mdx")
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("read the CLI reference: %v", err)
	}
	doc := string(data)
	fs := newFlagSet(&cliFlags{})

	fs.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		if !strings.Contains(doc, `flag: "--`+flag.Name+`"`) {
			t.Errorf("%s doesn't document --%s", page, flag.Name)
		}
		if flag.Shorthand != "" && !strings.Contains(doc, `short: "-`+flag.Shorthand+`", flag: "--`+flag.Name+`"`) {
			t.Errorf("%s doesn't give -%s as the short form of --%s", page, flag.Shorthand, flag.Name)
		}
	})

	for _, match := range regexp.MustCompile(`flag: "--([a-z0-9-]+)"`).FindAllStringSubmatch(doc, -1) {
		if fs.Lookup(match[1]) == nil {
			t.Errorf("%s documents --%s, which s3ctl doesn't accept", page, match[1])
		}
	}
}
