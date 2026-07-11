package spec

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

func TestEveryDaemonPathHasAnSDKSupportDecision(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pathLine := regexp.MustCompile(`^  (/[^:]+):$`)
	var paths []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if match := pathLine.FindStringSubmatch(scanner.Text()); match != nil {
			paths = append(paths, match[1])
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	decisions := map[string]string{
		"/v1/health":                 "supported",
		"/v1/ready":                  "unsupported: deployment readiness probe",
		"/v1/chat":                   "supported",
		"/v1/sessions":               "supported",
		"/v1/sessions/{id}":          "supported",
		"/v1/sessions/{id}/messages": "supported",
		"/v1/stats":                  "supported",
		"/v1/review":                 "unsupported: asynchronous review orchestration",
		"/v1/review/status":          "unsupported: review worker status",
	}
	sort.Strings(paths)
	var decided []string
	for path := range decisions {
		decided = append(decided, path)
	}
	sort.Strings(decided)
	if len(paths) != len(decided) {
		t.Fatalf("contract paths = %v, SDK decisions = %v", paths, decided)
	}
	for i := range paths {
		if paths[i] != decided[i] {
			t.Fatalf("contract paths = %v, SDK decisions = %v", paths, decided)
		}
	}
}
