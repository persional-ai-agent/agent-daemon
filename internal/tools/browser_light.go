package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type lightBrowserPage struct {
	URL       string
	HTML      string
	LoadedAt  string
	Status    int
	Truncated bool
	Headers   map[string]string
}

type lightBrowserState struct {
	stack      []lightBrowserPage
	formInputs map[string]string
	elements   map[string]lightElement
}

type lightElement struct {
	Kind  string // link|input|button
	Text  string
	Href  string
	Field string // for input
}

var lightBrowserMu sync.Mutex
var lightBrowserSessions = map[string]*lightBrowserState{}

func getLightBrowser(sessionID string) *lightBrowserState {
	lightBrowserMu.Lock()
	defer lightBrowserMu.Unlock()
	if sessionID == "" {
		sessionID = "default"
	}
	st := lightBrowserSessions[sessionID]
	if st == nil {
		st = &lightBrowserState{formInputs: map[string]string{}, elements: map[string]lightElement{}}
		lightBrowserSessions[sessionID] = st
	}
	return st
}

func (b *BuiltinTools) browserNavigate(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserNavigateCDP(ctx, args, tc)
	}
	url := strings.TrimSpace(strArg(args, "url"))
	if url == "" {
		return nil, errors.New("url required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(resp.Body)
	html := string(bs)
	truncated := false
	if len(html) > 300_000 {
		html = html[:300_000]
		truncated = true
	}
	p := lightBrowserPage{
		URL:       url,
		HTML:      html,
		LoadedAt:  time.Now().Format(time.RFC3339),
		Status:    resp.StatusCode,
		Truncated: truncated,
		Headers:   map[string]string{},
	}
	for k, v := range resp.Header {
		if len(v) > 0 {
			p.Headers[k] = v[0]
		}
	}
	st := getLightBrowser(tc.SessionID)
	st.stack = append(st.stack, p)
	st.formInputs = map[string]string{}
	st.elements = map[string]lightElement{}
	return map[string]any{
		"success":   true,
		"url":       url,
		"status":    resp.StatusCode,
		"loaded_at": p.LoadedAt,
		"truncated": truncated,
		"note":      "Lightweight browser: HTML fetched over HTTP; no JS execution.",
	}, nil
}

func (b *BuiltinTools) browserSnapshot(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserSnapshotCDP(ctx, args, tc)
	}
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	// Build a pseudo "accessibility snapshot" with lightweight ref IDs (@e1...).
	els := extractLightElements(p.HTML, 80)
	st.elements = map[string]lightElement{}
	var sb strings.Builder
	for i, el := range els {
		ref := fmt.Sprintf("@e%d", i+1)
		st.elements[ref] = el
		switch el.Kind {
		case "link":
			sb.WriteString("[" + ref + "] link: " + el.Text + " (href=" + el.Href + ")\n")
		case "input":
			sb.WriteString("[" + ref + "] input: " + el.Field + "\n")
		case "button":
			sb.WriteString("[" + ref + "] button: " + el.Text + "\n")
		}
	}
	maxChars := intArg(args, "max_chars", 120_000)
	if maxChars <= 0 {
		maxChars = 120_000
	}
	if maxChars > 300_000 {
		maxChars = 300_000
	}
	sb.WriteString("\n")
	sb.WriteString(htmlToTextLite(p.HTML, maxChars))
	return map[string]any{
		"success":   true,
		"url":       p.URL,
		"status":    p.Status,
		"loaded_at": p.LoadedAt,
		"content":   sb.String(),
		"pending_dialogs": []any{},
		"note":      "Lightweight snapshot: derived from fetched HTML; ref IDs are best-effort (no JS/DOM).",
	}, nil
}

func (b *BuiltinTools) browserBack(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) <= 1 {
		return map[string]any{"success": true, "can_go_back": false}, nil
	}
	st.stack = st.stack[:len(st.stack)-1]
	cur := st.stack[len(st.stack)-1]
	return map[string]any{"success": true, "can_go_back": len(st.stack) > 1, "url": cur.URL}, nil
}

func (b *BuiltinTools) browserClick(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserClickCDP(ctx, args, tc)
	}
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	if ref := strings.TrimSpace(strArg(args, "ref")); ref != "" {
		if st.elements != nil {
			if el, ok := st.elements[ref]; ok && el.Kind == "link" && strings.TrimSpace(el.Href) != "" {
				next, err := resolveURL(p.URL, el.Href)
				if err != nil {
					return nil, err
				}
				return b.browserNavigate(ctx, map[string]any{"url": next}, tc)
			}
		}
		return map[string]any{"success": false, "error": "ref not found or not a link"}, nil
	}
	targetText := strings.TrimSpace(strArg(args, "text"))
	hrefContains := strings.TrimSpace(strArg(args, "href_contains"))
	if targetText == "" && hrefContains == "" {
		return nil, errors.New("ref, text or href_contains required")
	}
	href, ok := findAnchorHref(p.HTML, targetText, hrefContains)
	if !ok {
		return map[string]any{"success": false, "error": "link not found"}, nil
	}
	next, err := resolveURL(p.URL, href)
	if err != nil {
		return nil, err
	}
	return b.browserNavigate(ctx, map[string]any{"url": next}, tc)
}

func (b *BuiltinTools) browserType(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserTypeCDP(ctx, args, tc)
	}
	// Lightweight browser has no DOM state; we accept the call for parity.
	field := strings.TrimSpace(strArg(args, "field"))
	if ref := strings.TrimSpace(strArg(args, "ref")); ref != "" {
		st := getLightBrowser(tc.SessionID)
		if st.elements != nil {
			if el, ok := st.elements[ref]; ok && el.Kind == "input" {
				if strings.TrimSpace(el.Field) != "" {
					field = el.Field
				}
			}
		}
	}
	text := strArg(args, "text")
	if field == "" {
		field = "unknown"
	}
	st := getLightBrowser(tc.SessionID)
	if st.formInputs == nil {
		st.formInputs = map[string]string{}
	}
	st.formInputs[field] = text
	return map[string]any{"success": true, "note": "Lightweight browser: stored typed values for best-effort GET form submission.", "field": field}, nil
}

func extractLightElements(html string, limit int) []lightElement {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	out := make([]lightElement, 0, limit)
	// Links
	linkRe := regexp.MustCompile(`(?is)<a[^>]+href\\s*=\\s*['"]([^'"]+)['"][^>]*>(.*?)</a>`)
	for _, m := range linkRe.FindAllStringSubmatch(html, limit) {
		href := strings.TrimSpace(m[1])
		txt := strings.TrimSpace(htmlToTextLite(m[2], 200))
		if txt == "" {
			txt = "(link)"
		}
		out = append(out, lightElement{Kind: "link", Text: txt, Href: href})
	}
	// Inputs
	inputRe := regexp.MustCompile(`(?is)<input[^>]*(?:name\\s*=\\s*['"]([^'"]+)['"])?[^>]*>`)
	for _, m := range inputRe.FindAllStringSubmatch(html, limit-len(out)) {
		field := strings.TrimSpace(m[1])
		if field == "" {
			field = "input"
		}
		out = append(out, lightElement{Kind: "input", Field: field})
		if len(out) >= limit {
			return out
		}
	}
	// Buttons (text only)
	btnRe := regexp.MustCompile(`(?is)<button[^>]*>(.*?)</button>`)
	for _, m := range btnRe.FindAllStringSubmatch(html, limit-len(out)) {
		txt := strings.TrimSpace(htmlToTextLite(m[1], 200))
		if txt == "" {
			txt = "(button)"
		}
		out = append(out, lightElement{Kind: "button", Text: txt})
		if len(out) >= limit {
			return out
		}
	}
	return out
}

func (b *BuiltinTools) browserScroll(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	direction := strings.ToLower(strings.TrimSpace(strArg(args, "direction")))
	if direction == "" {
		direction = "down"
	}
	switch direction {
	case "up", "down", "left", "right":
	default:
		direction = "down"
	}
	amount := intArg(args, "amount", 1)
	if amount <= 0 {
		amount = 1
	}
	if amount > 100 {
		amount = 100
	}
	return map[string]any{
		"success":           true,
		"direction":         direction,
		"amount":            amount,
		"scroll_performed":  false,
		"note":              "Lightweight browser: scroll is a no-op.",
	}, nil
}

func (b *BuiltinTools) browserPress(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserPressCDP(ctx, args, tc)
	}
	key := strings.TrimSpace(strArg(args, "key"))
	if key == "" {
		key = "unknown"
	}
	if strings.EqualFold(key, "enter") {
		st := getLightBrowser(tc.SessionID)
		if len(st.stack) == 0 {
			return nil, errors.New("no page loaded; call browser_navigate first")
		}
		p := st.stack[len(st.stack)-1]
		method, action, ok := findFirstForm(p.HTML)
		if !ok {
			return map[string]any{"success": true, "note": "Lightweight browser: no form found to submit.", "key": key}, nil
		}
		dest, err := resolveURL(p.URL, action)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(method, "get") || method == "" {
			u, err := url.Parse(dest)
			if err != nil {
				return nil, err
			}
			q := u.Query()
			for k, v := range st.formInputs {
				if strings.TrimSpace(k) == "" {
					continue
				}
				q.Set(k, v)
			}
			u.RawQuery = q.Encode()
			return b.browserNavigate(ctx, map[string]any{"url": u.String()}, tc)
		}
		return map[string]any{"success": false, "error": "Lightweight browser: only GET form submit supported.", "method": method}, nil
	}
	return map[string]any{"success": true, "note": "Lightweight browser: press is a no-op.", "key": key}, nil
}

func (b *BuiltinTools) browserGetImages(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	limit := intArg(args, "limit", 200)
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	srcs := findImageSrcs(p.HTML)
	resolved := make([]string, 0, len(srcs))
	for _, s := range srcs {
		if len(resolved) >= limit {
			break
		}
		u, err := resolveURL(p.URL, s)
		if err != nil {
			continue
		}
		resolved = append(resolved, u)
	}
	return map[string]any{"success": true, "url": p.URL, "images": resolved, "count": len(resolved), "applied_limit": limit, "note": "Lightweight browser: image list derived from <img src> only."}, nil
}

func (b *BuiltinTools) browserConsole(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserConsoleCDP(ctx, args, tc)
	}
	limit := intArg(args, "limit", 200)
	if limit <= 0 {
		limit = 200
	}
	// No JS execution -> no console logs.
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	return map[string]any{
		"success": true,
		"url":     p.URL,
		"logs":    []any{},
		"count":   0,
		"applied_limit": limit,
		"note":    "Lightweight browser: no JS execution, console logs are unavailable.",
	}, nil
}

func (b *BuiltinTools) browserDialog(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserDialogCDP(context.Background(), args, tc)
	}
	// No JS -> no dialogs.
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	return map[string]any{
		"success": true,
		"url":     p.URL,
		"dialog":  nil,
		"note":    "Lightweight browser: dialogs are unavailable (no JS execution).",
	}, nil
}

func (b *BuiltinTools) browserCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if cdpEnabled() {
		return b.browserCDPInfoCDP(ctx, args, tc)
	}
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	includeHTML := boolArg(args, "include_html", false)
	res := map[string]any{
		"success":   true,
		"url":       p.URL,
		"status":    p.Status,
		"loaded_at": p.LoadedAt,
		"headers":   p.Headers,
		"html_len":  len(p.HTML),
		"note":      "Lightweight browser: no CDP; returning fetched-page metadata only.",
	}
	if includeHTML {
		html := p.HTML
		if len(html) > 120_000 {
			html = html[:120_000]
		}
		res["html"] = html
	}
	return res, nil
}

func (b *BuiltinTools) browserVision(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	limit := intArg(args, "limit", 5)
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	st := getLightBrowser(tc.SessionID)
	if len(st.stack) == 0 {
		return nil, errors.New("no page loaded; call browser_navigate first")
	}
	p := st.stack[len(st.stack)-1]
	srcs := findImageSrcs(p.HTML)
	items := make([]map[string]any, 0)
	for _, s := range srcs {
		if len(items) >= limit {
			break
		}
		u, err := resolveURL(p.URL, s)
		if err != nil {
			continue
		}
		meta, err := fetchImageMeta(ctx, u)
		if err != nil {
			items = append(items, map[string]any{"url": u, "success": false, "error": err.Error()})
			continue
		}
		meta["url"] = u
		items = append(items, meta)
	}
	return map[string]any{
		"success": true,
		"url":     p.URL,
		"count":   len(items),
		"images":  items,
		"note":    "Lightweight browser: downloads <img src> and returns basic image metadata only.",
	}, nil
}

func (b *BuiltinTools) browserNotSupported(_ context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	return map[string]any{
		"success":   false,
		"error":     "not supported in lightweight browser implementation",
		"available": false,
		"hint":      "This action requires a real browser engine (JS/DOM).",
	}, nil
}

func htmlToTextLite(html string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 120_000
	}
	// Very small & safe: strip tags naively.
	s := html
	// Remove scripts/styles quickly.
	s = stripBetween(s, "<script", "</script>")
	s = stripBetween(s, "<style", "</style>")
	out := make([]rune, 0, len(s))
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
			out = append(out, ' ')
		default:
			if !inTag {
				out = append(out, r)
			}
		}
		if len(out) >= maxChars {
			break
		}
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	if len(text) > maxChars {
		text = text[:maxChars]
	}
	return text
}

func stripBetween(s, startToken, endToken string) string {
	lower := strings.ToLower(s)
	startLower := strings.ToLower(startToken)
	endLower := strings.ToLower(endToken)
	for {
		i := strings.Index(lower, startLower)
		if i < 0 {
			return s
		}
		j := strings.Index(lower[i:], endLower)
		if j < 0 {
			return s[:i]
		}
		j = i + j + len(endLower)
		s = s[:i] + s[j:]
		lower = strings.ToLower(s)
	}
}

var (
	reAnchor = regexp.MustCompile(`(?is)<a\b[^>]*\bhref\s*=\s*["']?([^"'\s>]+)[^>]*>(.*?)</a>`)
	reImg    = regexp.MustCompile(`(?is)<img\b[^>]*\bsrc\s*=\s*["']?([^"'\s>]+)`)
	reTags   = regexp.MustCompile(`(?is)<[^>]+>`)
)

func findAnchorHref(html, text, hrefContains string) (string, bool) {
	matches := reAnchor.FindAllStringSubmatch(html, 2000)
	for _, m := range matches {
		href := strings.TrimSpace(m[1])
		label := strings.TrimSpace(reTags.ReplaceAllString(m[2], " "))
		label = strings.Join(strings.Fields(label), " ")
		if hrefContains != "" && !strings.Contains(href, hrefContains) {
			continue
		}
		if text != "" && !strings.Contains(strings.ToLower(label), strings.ToLower(text)) {
			continue
		}
		if href != "" {
			return href, true
		}
	}
	return "", false
}

func findImageSrcs(html string) []string {
	matches := reImg.FindAllStringSubmatch(html, 2000)
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		src := strings.TrimSpace(m[1])
		if src == "" || seen[src] {
			continue
		}
		seen[src] = true
		out = append(out, src)
	}
	return out
}

func resolveURL(base, ref string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return "", err
	}
	if u.IsAbs() {
		return u.String(), nil
	}
	bu, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	return bu.ResolveReference(u).String(), nil
}

var reForm = regexp.MustCompile(`(?is)<form\\b([^>]*)>`)

func findFirstForm(html string) (method string, action string, ok bool) {
	m := reForm.FindStringSubmatch(html)
	if m == nil {
		return "", "", false
	}
	attrs := m[1]
	method = findAttr(attrs, "method")
	action = findAttr(attrs, "action")
	return method, action, true
}

func findAttr(attrs string, name string) string {
	re := regexp.MustCompile(`(?is)\\b` + regexp.QuoteMeta(name) + `\\s*=\\s*[\"']?([^\"'\\s>]+)`)
	m := re.FindStringSubmatch(attrs)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func fetchImageMeta(ctx context.Context, u string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}
	// Read up to 2MB (enough for headers + small images).
	const max = 2 << 20
	bs, _ := io.ReadAll(io.LimitReader(resp.Body, max))
	cfg, format, err := image.DecodeConfig(bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	size := int64(len(bs))
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if v, err := strconv.ParseInt(strings.TrimSpace(cl), 10, 64); err == nil && v > 0 {
			size = v
		}
	}
	return map[string]any{
		"success": true,
		"format":  format,
		"width":   cfg.Width,
		"height":  cfg.Height,
		"bytes":   size,
	}, nil
}
