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

type Operation struct {
	Path               string
	Method             string
	RequestRequired    map[string]struct{}
	ResponseCodes      map[string]struct{}
	Response200Req     map[string]struct{}
	RequestFieldTypes  map[string]string
	Response200Types   map[string]string
	ParameterTypes     map[string]string
	ParameterRequired  map[string]bool
	RequestFieldEnums  map[string]map[string]struct{}
	Response200Enums   map[string]map[string]struct{}
	ParameterEnums     map[string]map[string]struct{}
}

type SpecSummary struct {
	Operations map[string]Operation
	Root       map[string]any
}

type Report struct {
	Base        string   `json:"base"`
	Target      string   `json:"target"`
	Breaking    []string `json:"breaking"`
	NonBreaking []string `json:"non_breaking"`
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func parseEnum(schema map[string]any) map[string]struct{} {
	items := asSlice(schema["enum"])
	if len(items) == 0 {
		return nil
	}
	out := map[string]struct{}{}
	for _, it := range items {
		out[fmt.Sprintf("%v", it)] = struct{}{}
	}
	return out
}

func resolveRef(root map[string]any, ref string) map[string]any {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "#/") {
		return map[string]any{}
	}
	cur := any(root)
	for _, p := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, ok := cur.(map[string]any)
		if !ok {
			return map[string]any{}
		}
		cur = m[p]
	}
	out, _ := cur.(map[string]any)
	return out
}

func resolvedSchema(root map[string]any, schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{}
	}
	if ref, _ := schema["$ref"].(string); strings.TrimSpace(ref) != "" {
		return resolvedSchema(root, resolveRef(root, ref))
	}
	return schema
}

func collectSchemaFields(root map[string]any, schema map[string]any) (map[string]string, map[string]struct{}, map[string]map[string]struct{}) {
	types := map[string]string{}
	required := map[string]struct{}{}
	enums := map[string]map[string]struct{}{}

	var walk func(map[string]any)
	walk = func(s map[string]any) {
		s = resolvedSchema(root, s)
		for _, r := range asSlice(s["required"]) {
			if n, ok := r.(string); ok {
				required[n] = struct{}{}
			}
		}
		props := asMap(s["properties"])
		for n, pv := range props {
			p := resolvedSchema(root, asMap(pv))
			if t, _ := p["type"].(string); strings.TrimSpace(t) != "" {
				types[n] = t
			}
			if enum := parseEnum(p); len(enum) > 0 {
				enums[n] = enum
			}
		}
		for _, sub := range asSlice(s["allOf"]) {
			walk(asMap(sub))
		}
	}

	walk(schema)
	return types, required, enums
}

func readSpec(path string) (SpecSummary, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return SpecSummary{}, err
	}
	root := map[string]any{}
	if err := yaml.Unmarshal(bs, &root); err != nil {
		return SpecSummary{}, err
	}
	paths := asMap(root["paths"])
	out := SpecSummary{Operations: map[string]Operation{}, Root: root}
	for p, rawPath := range paths {
		pathObj := asMap(rawPath)
		for method, rawOp := range pathObj {
			m := strings.ToUpper(strings.TrimSpace(method))
			if m == "" || strings.HasPrefix(m, "X-") {
				continue
			}
			op := Operation{
				Path:            p,
				Method:          m,
				RequestRequired: map[string]struct{}{},
				ResponseCodes:   map[string]struct{}{},
				Response200Req:  map[string]struct{}{},
				RequestFieldTypes: map[string]string{},
				Response200Types:  map[string]string{},
				ParameterTypes:    map[string]string{},
				ParameterRequired: map[string]bool{},
				RequestFieldEnums: map[string]map[string]struct{}{},
				Response200Enums:  map[string]map[string]struct{}{},
				ParameterEnums:    map[string]map[string]struct{}{},
			}
			opObj := asMap(rawOp)
			for _, pv := range asSlice(opObj["parameters"]) {
				pm := asMap(pv)
				name, _ := pm["name"].(string)
				in, _ := pm["in"].(string)
				if strings.TrimSpace(name) == "" || strings.TrimSpace(in) == "" {
					continue
				}
				key := in + ":" + name
				required, _ := pm["required"].(bool)
				op.ParameterRequired[key] = required
				pschema := resolvedSchema(root, asMap(pm["schema"]))
				if t, _ := pschema["type"].(string); strings.TrimSpace(t) != "" {
					op.ParameterTypes[key] = t
				}
				if enum := parseEnum(pschema); len(enum) > 0 {
					op.ParameterEnums[key] = enum
				}
			}
			reqBody := asMap(opObj["requestBody"])
			content := asMap(reqBody["content"])
			appJSON := asMap(content["application/json"])
			reqSchema := asMap(appJSON["schema"])
			reqTypes, reqRequired, reqEnums := collectSchemaFields(root, reqSchema)
			for k := range reqRequired {
				op.RequestRequired[k] = struct{}{}
			}
			op.RequestFieldTypes = reqTypes
			op.RequestFieldEnums = reqEnums
			responses := asMap(opObj["responses"])
			for code, rawResp := range responses {
				op.ResponseCodes[code] = struct{}{}
				if code != "200" {
					continue
				}
				respObj := asMap(rawResp)
				respContent := asMap(respObj["content"])
				respJSON := asMap(respContent["application/json"])
				respSchema := asMap(respJSON["schema"])
				respTypes, respRequired, respEnums := collectSchemaFields(root, respSchema)
				for k := range respRequired {
					op.Response200Req[k] = struct{}{}
				}
				op.Response200Types = respTypes
				op.Response200Enums = respEnums
			}
			key := m + " " + p
			out.Operations[key] = op
		}
	}
	return out, nil
}

func setDiff(a, b map[string]struct{}) (removed, added []string) {
	for k := range a {
		if _, ok := b[k]; !ok {
			removed = append(removed, k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			added = append(added, k)
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	return removed, added
}

func compareTypes(scope, key, baseT, targetT string, breaking, nonBreaking *[]string) {
	if strings.TrimSpace(baseT) == "" && strings.TrimSpace(targetT) != "" {
		*nonBreaking = append(*nonBreaking, fmt.Sprintf("%s added typed field %s: %s", scope, key, targetT))
		return
	}
	if strings.TrimSpace(baseT) != "" && strings.TrimSpace(targetT) == "" {
		*breaking = append(*breaking, fmt.Sprintf("%s removed type info for %s (was %s)", scope, key, baseT))
		return
	}
	if baseT != "" && targetT != "" && baseT != targetT {
		*breaking = append(*breaking, fmt.Sprintf("%s changed type for %s: %s -> %s", scope, key, baseT, targetT))
	}
}

func compareEnumShrink(scope, key string, baseEnum, targetEnum map[string]struct{}, breaking, nonBreaking *[]string) {
	if len(baseEnum) == 0 || len(targetEnum) == 0 {
		return
	}
	removed, added := setDiff(baseEnum, targetEnum)
	for _, v := range removed {
		*breaking = append(*breaking, fmt.Sprintf("%s removed enum value for %s: %s", scope, key, v))
	}
	for _, v := range added {
		*nonBreaking = append(*nonBreaking, fmt.Sprintf("%s added enum value for %s: %s", scope, key, v))
	}
}

func writeReport(path string, report Report) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func buildReport(basePath, targetPath string, base, target SpecSummary) Report {
	breaking := make([]string, 0)
	nonBreaking := make([]string, 0)

	for key, b := range base.Operations {
		t, ok := target.Operations[key]
		if !ok {
			breaking = append(breaking, fmt.Sprintf("removed operation: %s", key))
			continue
		}
		rmResp, addResp := setDiff(b.ResponseCodes, t.ResponseCodes)
		for _, c := range rmResp {
			breaking = append(breaking, fmt.Sprintf("%s removed response code: %s", key, c))
		}
		for _, c := range addResp {
			nonBreaking = append(nonBreaking, fmt.Sprintf("%s added response code: %s", key, c))
		}

		rmReq, addReq := setDiff(b.RequestRequired, t.RequestRequired)
		for _, f := range rmReq {
			nonBreaking = append(nonBreaking, fmt.Sprintf("%s removed required request field: %s", key, f))
		}
		for _, f := range addReq {
			breaking = append(breaking, fmt.Sprintf("%s added required request field: %s", key, f))
		}

		rm200, add200 := setDiff(b.Response200Req, t.Response200Req)
		for _, f := range rm200 {
			breaking = append(breaking, fmt.Sprintf("%s removed 200 required response field: %s", key, f))
		}
		for _, f := range add200 {
			nonBreaking = append(nonBreaking, fmt.Sprintf("%s added 200 required response field: %s", key, f))
		}
		allReqKeys := map[string]struct{}{}
		for k := range b.RequestFieldTypes {
			allReqKeys[k] = struct{}{}
		}
		for k := range t.RequestFieldTypes {
			allReqKeys[k] = struct{}{}
		}
		for k := range allReqKeys {
			compareTypes(key+" request", k, b.RequestFieldTypes[k], t.RequestFieldTypes[k], &breaking, &nonBreaking)
			compareEnumShrink(key+" request", k, b.RequestFieldEnums[k], t.RequestFieldEnums[k], &breaking, &nonBreaking)
		}
		allRespKeys := map[string]struct{}{}
		for k := range b.Response200Types {
			allRespKeys[k] = struct{}{}
		}
		for k := range t.Response200Types {
			allRespKeys[k] = struct{}{}
		}
		for k := range allRespKeys {
			compareTypes(key+" response200", k, b.Response200Types[k], t.Response200Types[k], &breaking, &nonBreaking)
			compareEnumShrink(key+" response200", k, b.Response200Enums[k], t.Response200Enums[k], &breaking, &nonBreaking)
		}
		allParamKeys := map[string]struct{}{}
		for k := range b.ParameterTypes {
			allParamKeys[k] = struct{}{}
		}
		for k := range t.ParameterTypes {
			allParamKeys[k] = struct{}{}
		}
		for k := range b.ParameterRequired {
			allParamKeys[k] = struct{}{}
		}
		for k := range t.ParameterRequired {
			allParamKeys[k] = struct{}{}
		}
		for k := range allParamKeys {
			compareTypes(key+" parameter", k, b.ParameterTypes[k], t.ParameterTypes[k], &breaking, &nonBreaking)
			if b.ParameterRequired[k] != t.ParameterRequired[k] {
				if !b.ParameterRequired[k] && t.ParameterRequired[k] {
					breaking = append(breaking, fmt.Sprintf("%s parameter became required: %s", key, k))
				} else {
					nonBreaking = append(nonBreaking, fmt.Sprintf("%s parameter became optional: %s", key, k))
				}
			}
			compareEnumShrink(key+" parameter", k, b.ParameterEnums[k], t.ParameterEnums[k], &breaking, &nonBreaking)
		}
	}

	for key := range target.Operations {
		if _, ok := base.Operations[key]; !ok {
			nonBreaking = append(nonBreaking, fmt.Sprintf("added operation: %s", key))
		}
	}

	sort.Strings(breaking)
	sort.Strings(nonBreaking)
	return Report{
		Base:        basePath,
		Target:      targetPath,
		Breaking:    breaking,
		NonBreaking: nonBreaking,
	}
}

func main() {
	basePath := flag.String("base", "", "base openapi file")
	targetPath := flag.String("target", "", "target openapi file")
	ackPath := flag.String("ack", "", "breaking change acknowledgement file path")
	reportPath := flag.String("report", "artifacts/contract-diff.json", "json report output path")
	flag.Parse()

	if strings.TrimSpace(*basePath) == "" || strings.TrimSpace(*targetPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: contract_diff -base <file> -target <file> [-ack <file>]")
		os.Exit(2)
	}

	base, err := readSpec(*basePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read base spec failed: %v\n", err)
		os.Exit(2)
	}
	target, err := readSpec(*targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read target spec failed: %v\n", err)
		os.Exit(2)
	}

	report := buildReport(*basePath, *targetPath, base, target)
	if err := writeReport(*reportPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write report failed: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("== Contract Diff Report ==")
	fmt.Printf("Base:   %s\n", *basePath)
	fmt.Printf("Target: %s\n", *targetPath)
	if len(report.Breaking) == 0 {
		fmt.Println("Breaking changes: none")
	} else {
		fmt.Println("Breaking changes:")
		for _, it := range report.Breaking {
			fmt.Printf("- %s\n", it)
		}
	}
	if len(report.NonBreaking) == 0 {
		fmt.Println("Non-breaking changes: none")
	} else {
		fmt.Println("Non-breaking changes:")
		for _, it := range report.NonBreaking {
			fmt.Printf("- %s\n", it)
		}
	}

	if len(report.Breaking) == 0 {
		fmt.Printf("Report: %s\n", *reportPath)
		return
	}
	if strings.TrimSpace(*ackPath) == "" {
		fmt.Fprintln(os.Stderr, "breaking changes detected; acknowledgement file not provided")
		os.Exit(10)
	}
	if _, err := os.Stat(*ackPath); err != nil {
		fmt.Fprintf(os.Stderr, "breaking changes detected; missing acknowledgement file: %s\n", *ackPath)
		os.Exit(11)
	}
	fmt.Printf("breaking changes acknowledged by: %s\n", *ackPath)
	fmt.Printf("Report: %s\n", *reportPath)
}
