package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type replayCase struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	ContractPath string `json:"contract_path,omitempty"`
}

type coverageReport struct {
	OpenAPIFile   string            `json:"openapi_file"`
	ReplayFile    string            `json:"replay_file"`
	CoreTotal     int               `json:"core_total"`
	CoreCovered   int               `json:"core_covered"`
	CoreCoverage  float64           `json:"core_coverage"`
	Covered       []string          `json:"covered"`
	Uncovered     []string          `json:"uncovered"`
	ReplayOps     []string          `json:"replay_ops"`
	OperationCase map[string]string `json:"operation_case,omitempty"`
}

func covAsMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func loadOpenAPIOps(path string) (map[string]struct{}, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root := map[string]any{}
	if err := yaml.Unmarshal(bs, &root); err != nil {
		return nil, err
	}
	paths := covAsMap(root["paths"])
	out := map[string]struct{}{}
	for p, pv := range paths {
		pm := covAsMap(pv)
		for method := range pm {
			m := strings.ToUpper(strings.TrimSpace(method))
			if m == "" || strings.HasPrefix(m, "X-") {
				continue
			}
			op := m + " " + p
			if strings.HasPrefix(p, "/v1/ui/") || op == "POST /v1/chat" || op == "POST /v1/chat/cancel" {
				out[op] = struct{}{}
			}
		}
	}
	return out, nil
}

func loadReplayOps(path string) (map[string]struct{}, map[string]string, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var cases []replayCase
	if err := json.Unmarshal(bs, &cases); err != nil {
		return nil, nil, err
	}
	out := map[string]struct{}{}
	caseMap := map[string]string{}
	for _, c := range cases {
		p := strings.TrimSpace(c.ContractPath)
		if p == "" {
			p = strings.TrimSpace(c.Path)
		}
		m := strings.ToUpper(strings.TrimSpace(c.Method))
		if m == "" || p == "" {
			continue
		}
		op := m + " " + p
		out[op] = struct{}{}
		caseMap[op] = c.Method + " " + c.Path
	}
	return out, caseMap, nil
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func writeCoverageReport(path string, rep coverageReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func main() {
	openapi := flag.String("openapi", "docs/api/ui-chat-contract.openapi.yaml", "openapi file")
	replay := flag.String("replay", "internal/api/testdata/replay/cases.json", "replay cases file")
	report := flag.String("report", "artifacts/contract-coverage.json", "coverage report output")
	enforce := flag.Bool("enforce-core", true, "enforce core coverage=100%")
	flag.Parse()

	coreOps, err := loadOpenAPIOps(*openapi)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load openapi failed: %v\n", err)
		os.Exit(2)
	}
	replayOps, caseMap, err := loadReplayOps(*replay)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load replay failed: %v\n", err)
		os.Exit(2)
	}

	coveredSet := map[string]struct{}{}
	uncoveredSet := map[string]struct{}{}
	for op := range coreOps {
		if _, ok := replayOps[op]; ok {
			coveredSet[op] = struct{}{}
		} else {
			uncoveredSet[op] = struct{}{}
		}
	}
	coreTotal := len(coreOps)
	coreCovered := len(coveredSet)
	coverage := 100.0
	if coreTotal > 0 {
		coverage = float64(coreCovered) * 100.0 / float64(coreTotal)
	}
	rep := coverageReport{
		OpenAPIFile:   *openapi,
		ReplayFile:    *replay,
		CoreTotal:     coreTotal,
		CoreCovered:   coreCovered,
		CoreCoverage:  coverage,
		Covered:       sortedKeys(coveredSet),
		Uncovered:     sortedKeys(uncoveredSet),
		ReplayOps:     sortedKeys(replayOps),
		OperationCase: caseMap,
	}
	if err := writeCoverageReport(*report, rep); err != nil {
		fmt.Fprintf(os.Stderr, "write coverage report failed: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("== Contract Coverage Report ==")
	fmt.Printf("Core coverage: %.2f%% (%d/%d)\n", rep.CoreCoverage, rep.CoreCovered, rep.CoreTotal)
	fmt.Printf("Report: %s\n", *report)
	if len(rep.Uncovered) > 0 {
		fmt.Println("Uncovered core operations:")
		for _, op := range rep.Uncovered {
			fmt.Printf("- %s\n", op)
		}
	}
	if *enforce && len(rep.Uncovered) > 0 {
		fmt.Fprintln(os.Stderr, "core coverage gate failed: uncovered operations found")
		os.Exit(10)
	}
}
