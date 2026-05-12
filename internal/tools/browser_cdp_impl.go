package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

type cdpBrowserState struct {
	client   *cdpClient
	elements map[string]lightElement
	mu       sync.Mutex
	logs     []map[string]any
	dialog   map[string]any
}

var cdpBrowserMu sync.Mutex
var cdpBrowsers = map[string]*cdpBrowserState{}

func getCDPBrowser(ctx context.Context, sessionID string) (*cdpBrowserState, error) {
	if !cdpEnabled() {
		return nil, errors.New("cdp disabled (set env BROWSER_CDP_URL)")
	}
	if sessionID == "" {
		sessionID = "default"
	}
	cdpBrowserMu.Lock()
	st := cdpBrowsers[sessionID]
	cdpBrowserMu.Unlock()
	if st != nil && st.client != nil {
		return st, nil
	}

	c, err := dialCDP(ctx, strings.TrimSpace(getenv("BROWSER_CDP_URL")))
	if err != nil {
		return nil, err
	}
	st = &cdpBrowserState{client: c, elements: map[string]lightElement{}, logs: []map[string]any{}}
	c.mu.Lock()
	c.onEvent = func(method string, params json.RawMessage) {
		st.handleEvent(method, params)
	}
	c.mu.Unlock()

	// Enable minimal domains + log/dialog events.
	_ = c.call(ctx, "Page.enable", nil, nil)
	_ = c.call(ctx, "Runtime.enable", nil, nil)
	_ = c.call(ctx, "DOM.enable", nil, nil)
	_ = c.call(ctx, "Log.enable", nil, nil)
	cdpBrowserMu.Lock()
	cdpBrowsers[sessionID] = st
	cdpBrowserMu.Unlock()
	return st, nil
}

func (s *cdpBrowserState) handleEvent(method string, params json.RawMessage) {
	switch method {
	case "Runtime.consoleAPICalled":
		var p struct {
			Type      string  `json:"type"`
			Timestamp float64 `json:"timestamp"`
			Args      []struct {
				Value       json.RawMessage `json:"value"`
				Description string          `json:"description"`
				Type        string          `json:"type"`
			} `json:"args"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return
		}
		msg := ""
		for _, a := range p.Args {
			if msg != "" {
				msg += " "
			}
			if len(a.Value) > 0 {
				var v any
				if err := json.Unmarshal(a.Value, &v); err == nil {
					msg += fmt.Sprintf("%v", v)
					continue
				}
			}
			if strings.TrimSpace(a.Description) != "" {
				msg += a.Description
				continue
			}
			msg += a.Type
		}
		s.appendLog(map[string]any{
			"source":    "console",
			"level":     strings.TrimSpace(p.Type),
			"message":   msg,
			"timestamp": p.Timestamp,
		})
	case "Runtime.exceptionThrown":
		var p struct {
			Timestamp float64 `json:"timestamp"`
			ExceptionDetails struct {
				Text        string `json:"text"`
				URL         string `json:"url"`
				LineNumber  int    `json:"lineNumber"`
				ColumnNumber int   `json:"columnNumber"`
				Exception struct {
					Description string `json:"description"`
				} `json:"exception"`
			} `json:"exceptionDetails"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return
		}
		msg := strings.TrimSpace(p.ExceptionDetails.Text)
		if strings.TrimSpace(p.ExceptionDetails.Exception.Description) != "" {
			if msg != "" {
				msg += ": "
			}
			msg += p.ExceptionDetails.Exception.Description
		}
		s.appendLog(map[string]any{
			"source":    "exception",
			"level":     "error",
			"message":   msg,
			"url":       p.ExceptionDetails.URL,
			"line":      p.ExceptionDetails.LineNumber,
			"column":    p.ExceptionDetails.ColumnNumber,
			"timestamp": p.Timestamp,
		})
	case "Log.entryAdded":
		var p struct {
			Entry struct {
				Source    string  `json:"source"`
				Level     string  `json:"level"`
				Text      string  `json:"text"`
				URL       string  `json:"url"`
				LineNumber int    `json:"lineNumber"`
				Timestamp float64 `json:"timestamp"`
			} `json:"entry"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return
		}
		s.appendLog(map[string]any{
			"source":    p.Entry.Source,
			"level":     p.Entry.Level,
			"message":   p.Entry.Text,
			"url":       p.Entry.URL,
			"line":      p.Entry.LineNumber,
			"timestamp": p.Entry.Timestamp,
		})
	case "Page.javascriptDialogOpening":
		var p struct {
			Type             string `json:"type"`
			Message          string `json:"message"`
			DefaultPrompt    string `json:"defaultPrompt"`
			HasBrowserHandler bool  `json:"hasBrowserHandler"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return
		}
		s.mu.Lock()
		s.dialog = map[string]any{
			"type":            p.Type,
			"message":         p.Message,
			"default_prompt":  p.DefaultPrompt,
			"has_handler":     p.HasBrowserHandler,
		}
		s.mu.Unlock()
	case "Page.javascriptDialogClosed":
		s.mu.Lock()
		s.dialog = nil
		s.mu.Unlock()
	}
}

func (s *cdpBrowserState) appendLog(entry map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
	if len(s.logs) > 200 {
		s.logs = s.logs[len(s.logs)-200:]
	}
}

func (b *BuiltinTools) browserNavigateCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	targetURL := strings.TrimSpace(strArg(args, "url"))
	if targetURL == "" {
		return nil, errors.New("url required")
	}
	var res struct {
		FrameID string `json:"frameId"`
	}
	if err := st.client.call(ctx, "Page.navigate", map[string]any{"url": targetURL}, &res); err != nil {
		return nil, err
	}
	return map[string]any{
		"success": true,
		"url":     targetURL,
		"note":    "CDP browser: navigation requested (load events not awaited in minimal mode).",
	}, nil
}

func (b *BuiltinTools) browserSnapshotCDP(ctx context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	// Evaluate a snapshot script and return by value.
	script := `(function(){
  const out=[];
  const push=(o)=>{ out.push(o); };
  const links=[...document.querySelectorAll('a[href]')];
  for(const a of links){ if(out.length>=80) break; push({kind:'link', text:((a.innerText||a.textContent||'').trim()).slice(0,200), href:a.href}); }
  const inputs=[...document.querySelectorAll('input[name],textarea[name]')];
  for(const i of inputs){ if(out.length>=80) break; push({kind:'input', field:(i.name||'input').trim().slice(0,80)}); }
  const buttons=[...document.querySelectorAll('button')];
  for(const b of buttons){ if(out.length>=80) break; push({kind:'button', text:((b.innerText||b.textContent||'').trim()).slice(0,200)}); }
  const text=((document.body && (document.body.innerText||document.body.textContent))||'').toString();
  return {url: location.href, elements: out, text: text.slice(0,120000)};
})()`

	var evalRes struct {
		Result struct {
			Value struct {
				URL      string                   `json:"url"`
				Elements []map[string]any         `json:"elements"`
				Text     string                   `json:"text"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := st.client.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":      script,
		"returnByValue":   true,
		"awaitPromise":    true,
		"userGesture":     true,
	}, &evalRes); err != nil {
		return nil, err
	}

	st.elements = map[string]lightElement{}
	var sb strings.Builder
	for i, raw := range evalRes.Result.Value.Elements {
		if i >= 80 {
			break
		}
		kind, _ := raw["kind"].(string)
		ref := fmt.Sprintf("@e%d", i+1)
		switch kind {
		case "link":
			txt, _ := raw["text"].(string)
			href, _ := raw["href"].(string)
			el := lightElement{Kind: "link", Text: txt, Href: href}
			st.elements[ref] = el
			sb.WriteString("[" + ref + "] link: " + txt + " (href=" + href + ")\n")
		case "input":
			field, _ := raw["field"].(string)
			el := lightElement{Kind: "input", Field: field}
			st.elements[ref] = el
			sb.WriteString("[" + ref + "] input: " + field + "\n")
		case "button":
			txt, _ := raw["text"].(string)
			el := lightElement{Kind: "button", Text: txt}
			st.elements[ref] = el
			sb.WriteString("[" + ref + "] button: " + txt + "\n")
		}
	}
	sb.WriteString("\n")
	sb.WriteString(evalRes.Result.Value.Text)

	return map[string]any{
		"success":   true,
		"url":       evalRes.Result.Value.URL,
		"content":   sb.String(),
		"pending_dialogs": func() []any {
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.dialog == nil {
				return []any{}
			}
			return []any{st.dialog}
		}(),
		"note":      "CDP snapshot: derived from live DOM (JS executed).",
	}, nil
}

func (b *BuiltinTools) browserClickCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	ref := strings.TrimSpace(strArg(args, "ref"))
	text := strings.TrimSpace(strArg(args, "text"))
	hrefContains := strings.TrimSpace(strArg(args, "href_contains"))
	if ref == "" && text == "" && hrefContains == "" {
		return nil, errors.New("ref, text or href_contains required")
	}
	if ref != "" {
		el, ok := st.elements[ref]
		if !ok {
			return map[string]any{"success": false, "error": "ref not found"}, nil
		}
		if el.Kind != "link" && el.Kind != "button" {
			return map[string]any{"success": false, "error": "ref is not clickable"}, nil
		}
		// Reconstruct the same element list ordering and click by index.
		idx := refIndex(ref)
		if idx < 0 {
			return map[string]any{"success": false, "error": "invalid ref"}, nil
		}
		clickScript := fmt.Sprintf(`(function(){
  const out=[];
  const links=[...document.querySelectorAll('a[href]')];
  for(const a of links){ if(out.length>=80) break; out.push(a); }
  const inputs=[...document.querySelectorAll('input[name],textarea[name]')];
  for(const i of inputs){ if(out.length>=80) break; out.push(i); }
  const buttons=[...document.querySelectorAll('button')];
  for(const b of buttons){ if(out.length>=80) break; out.push(b); }
  const el=out[%d];
  if(!el) return {ok:false, error:'element missing'};
  el.click();
  return {ok:true};
})()`, idx)
		var resp struct {
			Result struct {
				Value map[string]any `json:"value"`
			} `json:"result"`
		}
		if err := st.client.call(ctx, "Runtime.evaluate", map[string]any{"expression": clickScript, "returnByValue": true, "awaitPromise": true, "userGesture": true}, &resp); err != nil {
			return nil, err
		}
		if ok, _ := resp.Result.Value["ok"].(bool); !ok {
			return map[string]any{"success": false, "error": fmt.Sprintf("%v", resp.Result.Value["error"])}, nil
		}
		return map[string]any{"success": true, "note": "CDP click dispatched."}, nil
	}

	// Fallback: click first matching link by text/href.
	needleText := strings.ToLower(text)
	needleHref := hrefContains
	clickScript := `(function(){
  const links=[...document.querySelectorAll('a[href]')];
  const t=` + "`" + jsonEscape(needleText) + "`" + `;
  const h=` + "`" + jsonEscape(needleHref) + "`" + `;
  for(const a of links){
    const label=((a.innerText||a.textContent||'').trim()).toLowerCase();
    const href=(a.href||'');
    if(h && href.indexOf(h)<0) continue;
    if(t && label.indexOf(t)<0) continue;
    a.click();
    return {ok:true, href: href};
  }
  return {ok:false, error:'link not found'};
})()`
	var resp struct {
		Result struct {
			Value map[string]any `json:"value"`
		} `json:"result"`
	}
	if err := st.client.call(ctx, "Runtime.evaluate", map[string]any{"expression": clickScript, "returnByValue": true, "awaitPromise": true, "userGesture": true}, &resp); err != nil {
		return nil, err
	}
	if ok, _ := resp.Result.Value["ok"].(bool); !ok {
		return map[string]any{"success": false, "error": fmt.Sprintf("%v", resp.Result.Value["error"])}, nil
	}
	return map[string]any{"success": true, "note": "CDP click dispatched.", "href": resp.Result.Value["href"]}, nil
}

func (b *BuiltinTools) browserTypeCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	field := strings.TrimSpace(strArg(args, "field"))
	if ref := strings.TrimSpace(strArg(args, "ref")); ref != "" {
		if el, ok := st.elements[ref]; ok && el.Kind == "input" && strings.TrimSpace(el.Field) != "" {
			field = el.Field
		}
	}
	text := strArg(args, "text")
	if strings.TrimSpace(field) == "" {
		return nil, errors.New("field required (or use ref for an input)")
	}

	script := `(function(){
  const name=` + "`" + jsonEscape(field) + "`" + `;
  const el=document.querySelector('input[name="'+name+'"],textarea[name="'+name+'"]');
  if(!el) return {ok:false, error:'input not found'};
  el.focus();
  el.value=` + "`" + jsonEscape(text) + "`" + `;
  el.dispatchEvent(new Event('input', {bubbles:true}));
  el.dispatchEvent(new Event('change', {bubbles:true}));
  return {ok:true};
})()`
	var resp struct {
		Result struct {
			Value map[string]any `json:"value"`
		} `json:"result"`
	}
	if err := st.client.call(ctx, "Runtime.evaluate", map[string]any{"expression": script, "returnByValue": true, "awaitPromise": true, "userGesture": true}, &resp); err != nil {
		return nil, err
	}
	if ok, _ := resp.Result.Value["ok"].(bool); !ok {
		return map[string]any{"success": false, "error": fmt.Sprintf("%v", resp.Result.Value["error"])}, nil
	}
	return map[string]any{"success": true, "field": field, "note": "CDP type: value set via DOM."}, nil
}

func (b *BuiltinTools) browserPressCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(strArg(args, "key"))
	if key == "" {
		key = "Enter"
	}
	// Minimal: only handle Enter by submitting the first form or clicking default button.
	if strings.EqualFold(key, "enter") {
		script := `(function(){
  const el=document.activeElement;
  if(el && el.form){ el.form.submit(); return {ok:true, via:'form.submit'}; }
  const f=document.querySelector('form');
  if(f){ f.submit(); return {ok:true, via:'form.submit'}; }
  const b=document.querySelector('button[type="submit"],input[type="submit"]');
  if(b){ b.click(); return {ok:true, via:'click'}; }
  return {ok:false, error:'no form found'};
})()`
		var resp struct {
			Result struct {
				Value map[string]any `json:"value"`
			} `json:"result"`
		}
		if err := st.client.call(ctx, "Runtime.evaluate", map[string]any{"expression": script, "returnByValue": true, "awaitPromise": true, "userGesture": true}, &resp); err != nil {
			return nil, err
		}
		if ok, _ := resp.Result.Value["ok"].(bool); !ok {
			return map[string]any{"success": false, "error": fmt.Sprintf("%v", resp.Result.Value["error"])}, nil
		}
		return map[string]any{"success": true, "note": "CDP press: enter dispatched.", "via": resp.Result.Value["via"]}, nil
	}
	return map[string]any{"success": false, "available": false, "error": "CDP press supports Enter only in minimal implementation"}, nil
}

func (b *BuiltinTools) browserCDPInfoCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	includeHTML := boolArg(args, "include_html", false)
	script := `(function(){
  const html = (document.documentElement && document.documentElement.outerHTML) ? document.documentElement.outerHTML : '';
  return {url: location.href, html: html.slice(0,120000), html_len: html.length};
})()`
	var evalRes struct {
		Result struct {
			Value struct {
				URL     string `json:"url"`
				HTML    string `json:"html"`
				HTMLLen int    `json:"html_len"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := st.client.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    script,
		"returnByValue": true,
		"awaitPromise":  true,
	}, &evalRes); err != nil {
		return nil, err
	}
	out := map[string]any{
		"success":   true,
		"url":       evalRes.Result.Value.URL,
		"html_len":  evalRes.Result.Value.HTMLLen,
		"note":      "CDP browser: returning live DOM metadata.",
	}
	if includeHTML {
		out["html"] = evalRes.Result.Value.HTML
	}
	return out, nil
}

func (b *BuiltinTools) browserConsoleCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	limit := intArg(args, "limit", 200)
	if limit <= 0 {
		limit = 200
	}
	st.mu.Lock()
	logs := st.logs
	st.logs = nil
	st.mu.Unlock()
	if logs == nil {
		logs = []map[string]any{}
	}
	if len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}
	out := make([]any, 0, len(logs))
	for _, l := range logs {
		out = append(out, l)
	}
	return map[string]any{
		"success":       true,
		"logs":          out,
		"count":         len(out),
		"applied_limit": limit,
		"note":          "CDP console: returns and clears buffered console/log/exception entries.",
	}, nil
}

func (b *BuiltinTools) browserDialogCDP(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	st, err := getCDPBrowser(ctx, tc.SessionID)
	if err != nil {
		return nil, err
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		// Default: just report current dialog if any.
		st.mu.Lock()
		d := st.dialog
		st.mu.Unlock()
		return map[string]any{"success": true, "dialog": d, "note": "CDP dialog: call again with action=accept|dismiss to respond."}, nil
	}
	accept := false
	switch action {
	case "accept":
		accept = true
	case "dismiss":
		accept = false
	default:
		return nil, errors.New("invalid action (expected accept or dismiss)")
	}
	promptText := strArg(args, "prompt_text")
	if err := st.client.call(ctx, "Page.handleJavaScriptDialog", map[string]any{
		"accept":     accept,
		"promptText": promptText,
	}, nil); err != nil {
		return nil, err
	}
	st.mu.Lock()
	st.dialog = nil
	st.mu.Unlock()
	return map[string]any{"success": true, "action": action, "note": "CDP dialog handled."}, nil
}

func refIndex(ref string) int {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "@e") {
		return -1
	}
	n := 0
	for i := 2; i < len(ref); i++ {
		if ref[i] < '0' || ref[i] > '9' {
			return -1
		}
		n = n*10 + int(ref[i]-'0')
	}
	if n <= 0 {
		return -1
	}
	return n - 1
}

func jsonEscape(s string) string {
	bs, _ := json.Marshal(s)
	// json.Marshal returns quoted string.
	if len(bs) >= 2 {
		return string(bs[1 : len(bs)-1])
	}
	return s
}

func getenv(key string) string { return strings.TrimSpace(os.Getenv(key)) }
