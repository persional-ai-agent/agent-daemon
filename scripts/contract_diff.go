package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Operation struct {
	Path            string
	Method          string
	RequestRequired map[string]struct{}
	ResponseCodes   map[string]struct{}
	Response200Req  map[string]struct{}
}

type SpecSummary struct {
	Operations map[string]Operation
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
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
	out := SpecSummary{Operations: map[string]Operation{}}
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
			}
			opObj := asMap(rawOp)
			reqBody := asMap(opObj["requestBody"])
			content := asMap(reqBody["content"])
			appJSON := asMap(content["application/json"])
			reqSchema := asMap(appJSON["schema"])
			for _, r := range asSlice(reqSchema["required"]) {
				if s, ok := r.(string); ok {
					op.RequestRequired[s] = struct{}{}
				}
			}
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
				for _, r := range asSlice(respSchema["required"]) {
					if s, ok := r.(string); ok {
						op.Response200Req[s] = struct{}{}
					}
				}
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

func main() {
	basePath := flag.String("base", "", "base openapi file")
	targetPath := flag.String("target", "", "target openapi file")
	ackPath := flag.String("ack", "", "breaking change acknowledgement file path")
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
	}

	for key := range target.Operations {
		if _, ok := base.Operations[key]; !ok {
			nonBreaking = append(nonBreaking, fmt.Sprintf("added operation: %s", key))
		}
	}

	sort.Strings(breaking)
	sort.Strings(nonBreaking)

	fmt.Println("== Contract Diff Report ==")
	fmt.Printf("Base:   %s\n", *basePath)
	fmt.Printf("Target: %s\n", *targetPath)
	if len(breaking) == 0 {
		fmt.Println("Breaking changes: none")
	} else {
		fmt.Println("Breaking changes:")
		for _, it := range breaking {
			fmt.Printf("- %s\n", it)
		}
	}
	if len(nonBreaking) == 0 {
		fmt.Println("Non-breaking changes: none")
	} else {
		fmt.Println("Non-breaking changes:")
		for _, it := range nonBreaking {
			fmt.Printf("- %s\n", it)
		}
	}

	if len(breaking) == 0 {
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
}
