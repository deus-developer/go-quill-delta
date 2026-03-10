package delta

import (
	"regexp"
	"strings"
)

// ============================================================
// Well-known attribute value constants from Quill editor
// ============================================================

// ListType values for the "list" attribute.
const (
	ListOrdered   = "ordered"
	ListBullet    = "bullet"
	ListChecked   = "checked"
	ListUnchecked = "unchecked"
)

// AlignType values for the "align" attribute.
const (
	AlignLeft    = "left"
	AlignCenter  = "center"
	AlignRight   = "right"
	AlignJustify = "justify"
)

// ScriptType values for the "script" attribute.
const (
	ScriptSub   = "sub"
	ScriptSuper = "super"
)

// DirectionType values for the "direction" attribute.
const (
	DirectionRTL = "rtl"
)

// ============================================================
// Precompiled regex patterns (matching quill-delta-to-html)
// ============================================================

var (
	ReHexColor   = regexp.MustCompile(`^#([0-9A-Fa-f]{6}|[0-9A-Fa-f]{3})$`)
	ReColorName  = regexp.MustCompile(`^[a-zA-Z]{1,50}$`)
	ReRGBColor   = regexp.MustCompile(`^rgb\((0|25[0-5]|2[0-4]\d|1\d\d|0?\d?\d),\s*(0|25[0-5]|2[0-4]\d|1\d\d|0?\d?\d),\s*(0|25[0-5]|2[0-4]\d|1\d\d|0?\d?\d)\)$`)
	ReFontName   = regexp.MustCompile(`^[a-zA-Z 0-9\-]{1,30}$`)
	ReSize       = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,20}$`)
	ReWidth      = regexp.MustCompile(`^\d*(px|em|%)?$`)
	ReTarget     = regexp.MustCompile(`^[_a-zA-Z0-9\-]{1,50}$`)
	ReRel        = regexp.MustCompile(`^[a-zA-Z\s\-]{1,250}$`)
	ReLang       = regexp.MustCompile(`^[a-zA-Z\s\-\\/+]{1,50}$`)
	ReMentionCls = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,500}$`)
	ReMentionID  = regexp.MustCompile(`^[a-zA-Z0-9_\-:.]{1,500}$`)

	// URL protocol whitelist — matches quill-delta-to-html url.sanitize
	ReURLSafe = regexp.MustCompile(`^((https?|s?ftp|file|blob|mailto|tel):|#|/|data:image/)`)
)

// ============================================================
// Validator functions — building blocks for user-defined rules
// ============================================================

// IsValidHexColor checks if s is a valid hex color (#RGB or #RRGGBB).
func IsValidHexColor(s string) bool { return ReHexColor.MatchString(s) }

// IsValidColorName checks if s is a valid CSS color name (alphabetic, 1-50 chars).
func IsValidColorName(s string) bool { return ReColorName.MatchString(s) }

// IsValidRGBColor checks if s is a valid rgb(r,g,b) color.
func IsValidRGBColor(s string) bool { return ReRGBColor.MatchString(s) }

// IsValidColor checks if s is any valid color (hex, name, or rgb).
func IsValidColor(s string) bool {
	return IsValidHexColor(s) || IsValidColorName(s) || IsValidRGBColor(s)
}

// IsValidFontName checks if s is a valid font name.
func IsValidFontName(s string) bool { return ReFontName.MatchString(s) }

// IsValidSize checks if s is a valid text size value.
func IsValidSize(s string) bool { return ReSize.MatchString(s) }

// IsValidWidth checks if s is a valid width value (number with optional px/em/%).
func IsValidWidth(s string) bool { return s != "" && ReWidth.MatchString(s) }

// IsValidTarget checks if s is a valid link target attribute.
func IsValidTarget(s string) bool { return ReTarget.MatchString(s) }

// IsValidRel checks if s is a valid link rel attribute.
func IsValidRel(s string) bool { return ReRel.MatchString(s) }

// IsValidLang checks if s is a valid code-block language.
func IsValidLang(s string) bool { return ReLang.MatchString(s) }

// IsValidList checks if s is a valid list type.
func IsValidList(s string) bool {
	return s == ListOrdered || s == ListBullet || s == ListChecked || s == ListUnchecked
}

// IsValidAlign checks if s is a valid alignment.
func IsValidAlign(s string) bool {
	return s == AlignLeft || s == AlignCenter || s == AlignRight || s == AlignJustify
}

// IsValidScript checks if s is a valid script type.
func IsValidScript(s string) bool {
	return s == ScriptSub || s == ScriptSuper
}

// IsValidDirection checks if s is a valid direction.
func IsValidDirection(s string) bool {
	return s == DirectionRTL
}

// IsValidHeader checks if n is a valid header level (1-6).
func IsValidHeader(n int) bool {
	return n >= 1 && n <= 6
}

// IsValidIndent checks if n is a valid indent level (1-30).
func IsValidIndent(n int) bool {
	return n >= 1 && n <= 30
}

// IsValidMentionClass checks if s is a valid mention CSS class.
func IsValidMentionClass(s string) bool { return ReMentionCls.MatchString(s) }

// IsValidMentionID checks if s is a valid mention ID.
func IsValidMentionID(s string) bool { return ReMentionID.MatchString(s) }

// IsValidMentionTarget checks if s is a valid mention target.
func IsValidMentionTarget(s string) bool {
	return s == "_self" || s == "_blank" || s == "_parent" || s == "_top"
}

// ============================================================
// URL utilities
// ============================================================

// SanitizeURL checks a URL against the protocol whitelist.
// Unsafe URLs are prefixed with "unsafe:".
// Users can provide their own URL sanitizer instead.
func SanitizeURL(url string) string {
	trimmed := strings.TrimLeft(url, " \t\n\r")
	if ReURLSafe.MatchString(trimmed) {
		return trimmed
	}
	return "unsafe:" + trimmed
}

// IsURLSafe returns true if the URL matches the safe protocol whitelist.
func IsURLSafe(url string) bool {
	return ReURLSafe.MatchString(strings.TrimLeft(url, " \t\n\r"))
}

// ============================================================
// Text utilities
// ============================================================

// CollapseNewlines replaces sequences of more than maxN consecutive newlines with exactly maxN.
func CollapseNewlines(s string, maxN int) string {
	if maxN <= 0 {
		return s
	}
	threshold := strings.Repeat("\n", maxN+1)
	replacement := strings.Repeat("\n", maxN)
	for strings.Contains(s, threshold) {
		s = strings.ReplaceAll(s, threshold, replacement)
	}
	return s
}

// ============================================================
// Delta inspection helpers
// ============================================================

// IsDocumentDelta checks if all ops are inserts (required for a document delta).
func IsDocumentDelta(d *Delta) bool {
	for i := range d.Ops {
		if !d.Ops[i].Insert.IsSet() {
			return false
		}
	}
	return true
}

// WalkAttributes calls fn for every attribute key-value pair across all ops.
// Return false from fn to stop iteration.
func WalkAttributes(d *Delta, fn func(opIndex int, key string, val AttrValue) bool) {
	for i := range d.Ops {
		for k, v := range d.Ops[i].Attributes {
			if !fn(i, k, v) {
				return
			}
		}
	}
}

// WalkEmbeds calls fn for every embed insert op.
// Return false from fn to stop iteration.
func WalkEmbeds(d *Delta, fn func(opIndex int, embed Embed, attrs AttributeMap) bool) {
	for i := range d.Ops {
		if d.Ops[i].Insert.IsEmbed() {
			if !fn(i, d.Ops[i].Insert.Embed(), d.Ops[i].Attributes) {
				return
			}
		}
	}
}

// TransformDelta applies a transformation function to each op, building a new delta.
// fn receives each op and returns the ops to include (zero, one, or many).
// This is the universal tool for custom sanitization pipelines.
func TransformDelta(d *Delta, fn func(op Op, index int) []Op) *Delta {
	result := New(nil)
	for i := range d.Ops {
		newOps := fn(d.Ops[i], i)
		for j := range newOps {
			result.push(newOps[j].clone())
		}
	}
	return result
}
