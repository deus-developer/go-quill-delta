package delta

import (
	"sort"
	"strconv"
	"strings"
)

// ============================================================
// Telegram entity types — no external dependencies
// ============================================================

// TelegramEntity represents a Telegram MessageEntity with UTF-16 offsets.
type TelegramEntity struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`            // UTF-16 code units
	Length        int    `json:"length"`            // UTF-16 code units
	URL           string `json:"url,omitempty"`     // text_link
	UserID        int64  `json:"user_id,omitempty"` // text_mention
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// Telegram entity type constants.
const (
	TGBold                 = "bold"
	TGItalic               = "italic"
	TGUnderline            = "underline"
	TGStrikethrough        = "strikethrough"
	TGSpoiler              = "spoiler"
	TGCode                 = "code"
	TGPre                  = "pre"
	TGTextLink             = "text_link"
	TGTextMention          = "text_mention"
	TGBlockquote           = "blockquote"
	TGExpandableBlockquote = "expandable_blockquote"
	TGCustomEmoji          = "custom_emoji"
	TGMention              = "mention"
	TGHashtag              = "hashtag"
	TGCashtag              = "cashtag"
	TGBotCommand           = "bot_command"
	TGURL                  = "url"
	TGEmail                = "email"
	TGPhoneNumber          = "phone_number"
)

// ============================================================
// Delta → Telegram (text + entities)
// ============================================================

// ToTelegram extracts plain text and Telegram entities from a document delta.
// Returns the plain text and a sorted slice of entities with UTF-16 offsets.
func ToTelegram(d *Delta) (string, []TelegramEntity) {
	// First pass: compute total text length
	totalRunes := 0
	for i := range d.Ops {
		if !d.Ops[i].Insert.IsSet() {
			continue
		}
		if d.Ops[i].Insert.IsText() {
			totalRunes += len([]rune(d.Ops[i].Insert.Text()))
		}
		// embeds skipped for Telegram (no embed representation)
	}

	var textBuf strings.Builder
	textBuf.Grow(totalRunes) // approximate

	var entities []TelegramEntity
	utf16Offset := 0

	for i := range d.Ops {
		if !d.Ops[i].Insert.IsSet() {
			continue
		}

		if d.Ops[i].Insert.IsEmbed() {
			// Skip embeds — Telegram doesn't support inline embeds in text
			continue
		}

		text := d.Ops[i].Insert.Text()
		if text == "" {
			continue
		}

		utf16Len := utf16RuneLen(text)

		// Extract entities from attributes
		if len(d.Ops[i].Attributes) > 0 {
			entities = appendTelegramEntities(entities, d.Ops[i].Attributes, utf16Offset, utf16Len)
		}

		textBuf.WriteString(text)
		utf16Offset += utf16Len
	}

	return textBuf.String(), entities
}

// appendTelegramEntities appends entities for the given attributes at the given offset.
func appendTelegramEntities(entities []TelegramEntity, attrs AttributeMap, offset, length int) []TelegramEntity {
	if b, ok := attrs.GetBool("bold"); ok && b {
		entities = append(entities, TelegramEntity{Type: TGBold, Offset: offset, Length: length})
	}
	if b, ok := attrs.GetBool("italic"); ok && b {
		entities = append(entities, TelegramEntity{Type: TGItalic, Offset: offset, Length: length})
	}
	if b, ok := attrs.GetBool("underline"); ok && b {
		entities = append(entities, TelegramEntity{Type: TGUnderline, Offset: offset, Length: length})
	}
	if b, ok := attrs.GetBool("strike"); ok && b {
		entities = append(entities, TelegramEntity{Type: TGStrikethrough, Offset: offset, Length: length})
	}
	if b, ok := attrs.GetBool("code"); ok && b {
		entities = append(entities, TelegramEntity{Type: TGCode, Offset: offset, Length: length})
	}
	if link, ok := attrs.GetString("link"); ok {
		entities = append(entities, TelegramEntity{Type: TGTextLink, Offset: offset, Length: length, URL: link})
	}
	return entities
}

// ============================================================
// Delta → Telegram (block-aware: handles code blocks, blockquotes, lists)
// ============================================================

// ToTelegramFull converts a document delta to Telegram text + entities,
// handling block-level formatting (code blocks, blockquotes).
// Lists are rendered as "• item" / "1. item" text.
func ToTelegramFull(d *Delta) (string, []TelegramEntity) {
	var textBuf strings.Builder
	var entities []TelegramEntity
	utf16Offset := 0
	orderedCounter := 0

	d.EachLine(func(line *Delta, lineAttrs AttributeMap, idx int) bool {
		lineStart := utf16Offset

		// List prefix
		if l, ok := lineAttrs.GetString("list"); ok {
			var prefix string
			indent := 0
			if n, ok := lineAttrs.GetNumber("indent"); ok {
				indent = int(n)
			}
			indentStr := strings.Repeat("  ", indent)

			switch l {
			case ListOrdered:
				orderedCounter++
				prefix = indentStr + strconv.Itoa(orderedCounter) + ". "
			case ListBullet:
				prefix = indentStr + "• "
				orderedCounter = 0
			case ListChecked:
				prefix = indentStr + "☑ "
				orderedCounter = 0
			case ListUnchecked:
				prefix = indentStr + "☐ "
				orderedCounter = 0
			}
			if prefix != "" {
				textBuf.WriteString(prefix)
				utf16Offset += utf16RuneLen(prefix)
			}
		} else {
			orderedCounter = 0
		}

		// Line content
		for j := range line.Ops {
			if !line.Ops[j].Insert.IsSet() || !line.Ops[j].Insert.IsText() {
				continue
			}
			text := line.Ops[j].Insert.Text()
			if text == "" {
				continue
			}
			utf16Len := utf16RuneLen(text)

			if len(line.Ops[j].Attributes) > 0 {
				entities = appendTelegramEntities(entities, line.Ops[j].Attributes, utf16Offset, utf16Len)
			}

			textBuf.WriteString(text)
			utf16Offset += utf16Len
		}

		// Newline
		textBuf.WriteByte('\n')
		utf16Offset++

		lineLen := utf16Offset - lineStart

		// Block-level entities
		if lineAttrs.Has("code-block") {
			lang := ""
			if s, ok := lineAttrs.GetString("code-block"); ok {
				lang = s
			}
			entities = append(entities, TelegramEntity{
				Type: TGPre, Offset: lineStart, Length: lineLen,
				Language: lang,
			})
		}
		if b, ok := lineAttrs.GetBool("blockquote"); ok && b {
			entities = append(entities, TelegramEntity{
				Type: TGBlockquote, Offset: lineStart, Length: lineLen,
			})
		}

		return true
	}, "\n")

	return textBuf.String(), entities
}

// ============================================================
// Telegram → Delta
// ============================================================

// FromTelegram converts Telegram text + entities into a Delta.
// Entities use UTF-16 offsets as per Telegram API.
func FromTelegram(text string, entities []TelegramEntity) *Delta {
	if text == "" {
		return New(nil)
	}

	if len(entities) == 0 {
		d := New(nil)
		d.Insert(text, nil)
		return d
	}

	// Build UTF-16 offset → byte offset mapping via a single pass over text.
	// We also collect boundary points in UTF-16 space.
	runes := []rune(text)
	utf16Len := 0
	for _, r := range runes {
		if r >= 0x10000 {
			utf16Len += 2
		} else {
			utf16Len++
		}
	}

	// Sort entities by offset, then by length descending (outer first)
	sorted := make([]TelegramEntity, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Offset != sorted[j].Offset {
			return sorted[i].Offset < sorted[j].Offset
		}
		return sorted[i].Length > sorted[j].Length
	})

	// Collect boundary points (deduped) using a compact approach
	pointSet := make(map[int]struct{}, len(sorted)*2+2)
	pointSet[0] = struct{}{}
	pointSet[utf16Len] = struct{}{}
	for _, ent := range sorted {
		end := ent.Offset + ent.Length
		if end > utf16Len {
			end = utf16Len
		}
		pointSet[ent.Offset] = struct{}{}
		pointSet[end] = struct{}{}
	}

	points := make([]int, 0, len(pointSet))
	for p := range pointSet {
		points = append(points, p)
	}
	sort.Ints(points)

	// Build UTF-16 offset → rune index mapping for boundary points only.
	// Single pass over runes, tracking UTF-16 position.
	utf16ToRune := make(map[int]int, len(points))
	u16pos := 0
	runeIdx := 0
	pointIdx := 0
	for pointIdx < len(points) && points[pointIdx] == 0 {
		utf16ToRune[0] = 0
		pointIdx++
	}
	for _, r := range runes {
		if r >= 0x10000 {
			u16pos += 2
		} else {
			u16pos++
		}
		runeIdx++
		for pointIdx < len(points) && points[pointIdx] == u16pos {
			utf16ToRune[u16pos] = runeIdx
			pointIdx++
		}
	}

	// Pre-compute entity start/end in UTF-16 for fast lookup
	type entSpan struct {
		start, end int
		entity     TelegramEntity
	}
	spans := make([]entSpan, len(sorted))
	for i, ent := range sorted {
		end := ent.Offset + ent.Length
		if end > utf16Len {
			end = utf16Len
		}
		spans[i] = entSpan{start: ent.Offset, end: end, entity: ent}
	}

	d := New(nil)

	for i := 0; i < len(points)-1; i++ {
		segStart := points[i]
		segEnd := points[i+1]
		if segStart >= segEnd || segStart >= utf16Len {
			continue
		}
		if segEnd > utf16Len {
			segEnd = utf16Len
		}

		runeStart := utf16ToRune[segStart]
		runeEnd := utf16ToRune[segEnd]
		segText := string(runes[runeStart:runeEnd])
		if segText == "" {
			continue
		}

		// Find all entities that cover this segment
		var attrs AttributeMap
		for _, sp := range spans {
			if sp.start > segStart {
				break // spans are sorted by start; no more can cover
			}
			if sp.start <= segStart && sp.end >= segEnd {
				if attrs == nil {
					attrs = make(AttributeMap, 2)
				}
				applyTelegramEntity(&attrs, sp.entity)
			}
		}

		d.Insert(segText, attrs)
	}

	return d
}

// applyTelegramEntity applies a Telegram entity's formatting to an AttributeMap.
func applyTelegramEntity(attrs *AttributeMap, ent TelegramEntity) {
	switch ent.Type {
	case TGBold:
		(*attrs)["bold"] = BoolAttr(true)
	case TGItalic:
		(*attrs)["italic"] = BoolAttr(true)
	case TGUnderline:
		(*attrs)["underline"] = BoolAttr(true)
	case TGStrikethrough:
		(*attrs)["strike"] = BoolAttr(true)
	case TGCode:
		(*attrs)["code"] = BoolAttr(true)
	case TGPre:
		if ent.Language != "" {
			(*attrs)["code-block"] = StringAttr(ent.Language)
		} else {
			(*attrs)["code-block"] = BoolAttr(true)
		}
	case TGTextLink:
		(*attrs)["link"] = StringAttr(ent.URL)
	case TGTextMention:
		(*attrs)["link"] = StringAttr("tg://user?id=" + strconv.FormatInt(ent.UserID, 10))
	case TGBlockquote, TGExpandableBlockquote:
		(*attrs)["blockquote"] = BoolAttr(true)
	case TGSpoiler:
		(*attrs)["spoiler"] = BoolAttr(true)
	case TGCustomEmoji:
		(*attrs)["custom-emoji"] = StringAttr(ent.CustomEmojiID)
	}
}

// ============================================================
// UTF-16 helpers — zero allocation for ASCII-only text
// ============================================================

// utf16RuneLen returns the number of UTF-16 code units needed to encode s.
func utf16RuneLen(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2 // surrogate pair
		} else {
			n++
		}
	}
	return n
}
