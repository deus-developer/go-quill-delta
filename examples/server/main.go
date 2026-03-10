package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	delta "github.com/deus-developer/go-quill-delta"
)

const maxOps = 10000
const maxDocLength = 100000

// Only these attributes are allowed — basic text formatting + links to trusted domains.
var allowedAttributes = map[string]bool{
	"bold":       true,
	"italic":     true,
	"underline":  true,
	"strike":     true,
	"header":     true,
	"list":       true,
	"indent":     true,
	"align":      true,
	"blockquote": true,
	"code-block": true,
	"code":       true,
	"script":     true,
	"direction":  true,
	"color":      true,
	"background": true,
	"font":       true,
	"size":       true,
	"link":       true,
}

var trustedDomains = []string{
	"ton.place",
	"tonplace.host",
	"tonplace.net",
}

func isTrustedDomain(host string) bool {
	host = strings.ToLower(host)
	for _, domain := range trustedDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

// sanitizeLink returns a safe link URL:
//   - trusted domains (https only) pass through as-is
//   - other https links are wrapped through api.tonplace.net/away?url=
//   - everything else (javascript:, data:, http:, etc.) is dropped
func sanitizeLink(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	if strings.ToLower(parsed.Scheme) != "https" {
		return ""
	}

	if isTrustedDomain(parsed.Hostname()) {
		return raw
	}

	away := &url.URL{
		Scheme:   "https",
		Host:     "api.tonplace.net",
		Path:     "/away",
		RawQuery: url.Values{"url": {raw}}.Encode(),
	}
	return away.String()
}

func isAllowedColor(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if s[0] == '#' {
		s = s[1:]
		if len(s) != 3 && len(s) != 6 {
			return false
		}
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return len(s) <= 30
}

// validateAttributeValue checks that a value is safe for a given attribute key.
func validateAttributeValue(key string, val delta.AttrValue) (delta.AttrValue, bool) {
	switch key {
	case "bold", "italic", "underline", "strike", "code", "blockquote", "code-block":
		if val.IsBool() {
			return val, true
		}
		return delta.AttrValue{}, false

	case "header":
		if val.IsNumber() {
			n := val.NumberVal()
			if n >= 1 && n <= 6 {
				return val, true
			}
		}
		return delta.AttrValue{}, false

	case "indent":
		if val.IsNumber() {
			n := val.NumberVal()
			if n >= 1 && n <= 8 {
				return val, true
			}
		}
		return delta.AttrValue{}, false

	case "list":
		if val.IsString() {
			switch val.StringVal() {
			case "ordered", "bullet", "checked", "unchecked":
				return val, true
			}
		}
		return delta.AttrValue{}, false

	case "align":
		if val.IsString() {
			switch val.StringVal() {
			case "center", "right", "justify":
				return val, true
			}
		}
		return delta.AttrValue{}, false

	case "direction":
		if val.IsString() && val.StringVal() == "rtl" {
			return val, true
		}
		return delta.AttrValue{}, false

	case "script":
		if val.IsString() {
			s := val.StringVal()
			if s == "sub" || s == "super" {
				return val, true
			}
		}
		return delta.AttrValue{}, false

	case "color", "background":
		if val.IsString() && isAllowedColor(val.StringVal()) {
			return val, true
		}
		return delta.AttrValue{}, false

	case "font":
		if val.IsString() {
			s := val.StringVal()
			if len(s) > 50 {
				return delta.AttrValue{}, false
			}
			for _, c := range s {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == ' ' || c == '-') {
					return delta.AttrValue{}, false
				}
			}
			return val, true
		}
		return delta.AttrValue{}, false

	case "size":
		if val.IsString() {
			switch val.StringVal() {
			case "small", "large", "huge":
				return val, true
			}
		}
		return delta.AttrValue{}, false

	case "link":
		if val.IsString() {
			safe := sanitizeLink(val.StringVal())
			if safe != "" {
				return delta.StringAttr(safe), true
			}
		}
		return delta.AttrValue{}, false
	}

	return delta.AttrValue{}, false
}

func validateDelta(d *delta.Delta) error {
	if len(d.Ops) == 0 {
		return fmt.Errorf("delta has no ops")
	}
	if len(d.Ops) > maxOps {
		return fmt.Errorf("too many ops: %d (max %d)", len(d.Ops), maxOps)
	}

	totalLength := 0
	for i, op := range d.Ops {
		if op.Insert.IsEmbed() {
			return fmt.Errorf("op[%d]: embeds are not allowed", i)
		}
		if op.Retain.IsEmbed() {
			return fmt.Errorf("op[%d]: embed retains are not allowed", i)
		}

		fields := 0
		if op.Insert.IsSet() {
			fields++
		}
		if op.Delete > 0 {
			fields++
		}
		if op.Retain.IsSet() {
			fields++
		}
		if fields != 1 {
			return fmt.Errorf("op[%d]: must have exactly one of insert/delete/retain, got %d", i, fields)
		}

		if op.Insert.IsText() {
			text := op.Insert.Text()
			if len(text) == 0 {
				return fmt.Errorf("op[%d]: insert text is empty", i)
			}
			if !utf8.ValidString(text) {
				return fmt.Errorf("op[%d]: insert text is not valid UTF-8", i)
			}
			if strings.ContainsRune(text, 0) {
				return fmt.Errorf("op[%d]: insert text contains null bytes", i)
			}
		}

		totalLength += op.Len()
		if totalLength > maxDocLength {
			return fmt.Errorf("delta too large: length exceeds %d", maxDocLength)
		}
	}

	return nil
}

func sanitizeAttributes(attrs delta.AttributeMap) delta.AttributeMap {
	if attrs == nil {
		return nil
	}

	clean := make(delta.AttributeMap, len(attrs))
	for key, val := range attrs {
		if !allowedAttributes[key] {
			continue
		}
		if safe, ok := validateAttributeValue(key, val); ok {
			clean[key] = safe
		}
	}

	if len(clean) == 0 {
		return nil
	}
	return clean
}

func sanitizeDelta(d *delta.Delta) *delta.Delta {
	result := delta.New(nil)
	for _, op := range d.Ops {
		op.Attributes = sanitizeAttributes(op.Attributes)
		result.Push(op)
	}
	result.Chop()
	return result
}

func handleDelta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var d delta.Delta
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	if err := validateDelta(&d); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	clean := sanitizeDelta(&d)
	writeJSON(w, http.StatusOK, clean)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	http.HandleFunc("/delta", handleDelta)
	addr := ":8080"
	fmt.Printf("listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
