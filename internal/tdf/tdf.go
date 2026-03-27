// Package tdf provides a parser and renderer for TDF (TheDraw Font) files.
// TDF is a binary font format from the BBS era used to create ANSI block-letter
// art. Each font file contains a 20-byte magic header, a 4-byte ID marker, and
// one font record with a 209-byte definition block (name, type, spacing,
// blocksize, 94-entry char offset table) followed by variable-length char data.
package tdf

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Magic is the TDF file signature (20 bytes: 0x13 + 18 ASCII chars + 0x1A).
const Magic = "\x13TheDraw FONTS file\x1a"

// ErrNotTDF is returned when a file is not a valid TDF font.
var ErrNotTDF = errors.New("not a valid TDF font file")

// cell is a single rendered cell in a TDF character row.
type cell struct {
	char byte // CP437 glyph byte
	attr byte // color attribute (bg<<4 | fg) for Color fonts; 0x07 for others
}

// Font represents a parsed TDF font.
type Font struct {
	Name      string
	Type      byte   // 0=Outline, 1=Block, 2=Color
	Spacing   byte
	BlockSize uint16
	// chars maps ASCII byte (33–126) → rows of cells.
	chars map[byte][][]cell
}

// Parse parses a TDF font file from raw bytes, returning the first (and
// typically only) font in the file. Returns ErrNotTDF if the magic or
// identification marker are absent.
func Parse(data []byte) (*Font, error) {
	if len(data) < 20 || string(data[:20]) != Magic {
		return nil, ErrNotTDF
	}
	pos := 20 // skip magic

	// File identification marker: 55 AA 00 FF
	if pos+4 > len(data) ||
		data[pos] != 0x55 || data[pos+1] != 0xAA ||
		data[pos+2] != 0x00 || data[pos+3] != 0xFF {
		return nil, fmt.Errorf("tdf: missing identification marker")
	}
	pos += 4 // pos = 24

	// Name: 1-byte length prefix + 12-byte null-padded buffer = 13 bytes.
	if pos+13 > len(data) {
		return nil, fmt.Errorf("tdf: truncated at name")
	}
	nameLen := int(data[pos])
	if nameLen > 12 {
		nameLen = 12
	}
	pos++ // pos = 25
	nameBytes := data[pos : pos+nameLen]
	// Trim at first null in case nameLen overshoots actual content.
	for i, b := range nameBytes {
		if b == 0 {
			nameBytes = nameBytes[:i]
			break
		}
	}
	name := string(nameBytes)
	pos += 12 // skip full 12-byte name buffer → pos = 37

	// Reserved: 4 bytes (always zero in observed files).
	if pos+4 > len(data) {
		return nil, fmt.Errorf("tdf: truncated at reserved bytes")
	}
	pos += 4 // pos = 41

	// Font type, letter spacing, block size (total char data bytes).
	if pos+4 > len(data) {
		return nil, fmt.Errorf("tdf: truncated at metadata")
	}
	fontType := data[pos]
	pos++
	spacing := data[pos]
	pos++
	blockSize := uint16(data[pos]) | uint16(data[pos+1])<<8
	pos += 2 // pos = 45

	// Character offset table: 94 entries × 2 bytes LE for ASCII 33–126.
	// A value of 0xFFFF means the character is undefined in this font.
	if pos+188 > len(data) {
		return nil, fmt.Errorf("tdf: truncated at offset table")
	}
	offsets := make([]uint16, 94)
	for i := range offsets {
		offsets[i] = uint16(data[pos]) | uint16(data[pos+1])<<8
		pos += 2
	}
	// pos = 233 — character data section starts here.
	charDataBase := pos
	_ = blockSize // used for documentation only; actual end is determined by 0x00 terminators

	f := &Font{
		Name:      name,
		Type:      fontType,
		Spacing:   spacing,
		BlockSize: blockSize,
		chars:     make(map[byte][][]cell),
	}

	// Color fonts use 2 bytes per cell (char, attr); others use 1 byte.
	bytesPerCell := 1
	if fontType == 2 {
		bytesPerCell = 2
	}

	for i, off := range offsets {
		if off == 0xFFFF {
			continue // undefined character
		}
		ascii := byte(33 + i)
		startOff := charDataBase + int(off)
		if startOff >= len(data) {
			continue
		}
		rows := parseCharRows(data[startOff:], bytesPerCell)
		if len(rows) > 0 {
			f.chars[ascii] = rows
		}
	}

	return f, nil
}

// parseCharRows parses variable-length TDF character data into rows of cells.
//   - Each row ends with 0x0D (CR).
//   - The character (and final row) ends with 0x00 (NUL).
//   - bytesPerCell is 2 for Color fonts, 1 for Block/Outline.
func parseCharRows(data []byte, bytesPerCell int) [][]cell {
	var rows [][]cell
	var cur []cell
	pos := 0
	for pos < len(data) {
		b := data[pos]
		if b == 0x00 {
			// End of character — append last row if non-empty.
			if len(cur) > 0 {
				rows = append(rows, cur)
			}
			break
		}
		if b == 0x0D {
			// End of row.
			rows = append(rows, cur)
			cur = nil
			pos++
			continue
		}
		if bytesPerCell == 2 {
			if pos+1 >= len(data) {
				break
			}
			cur = append(cur, cell{char: data[pos], attr: data[pos+1]})
			pos += 2
		} else {
			cur = append(cur, cell{char: data[pos], attr: 0x07})
			pos++
		}
	}
	return rows
}

// Height returns the maximum row count across all defined characters.
func (f *Font) Height() int {
	max := 0
	for _, rows := range f.chars {
		if len(rows) > max {
			max = len(rows)
		}
	}
	if max == 0 {
		return 1
	}
	return max
}

// charWidth returns the visible column width of the widest row for ascii.
func (f *Font) charWidth(ascii byte) int {
	rows := f.chars[ascii]
	w := 0
	for _, row := range rows {
		if len(row) > w {
			w = len(row)
		}
	}
	return w
}

// MeasureWidth returns the total rendered column width of text using this font.
func (f *Font) MeasureWidth(text string) int {
	runes := []rune(text)
	total := 0
	sp := int(f.Spacing)
	for i, ch := range runes {
		w := f.charWidth(byte(ch))
		if w == 0 {
			w = 1 // minimum 1-column placeholder for undefined chars
		}
		total += w
		if i < len(runes)-1 {
			total += sp
		}
	}
	return total
}

// Render renders text as multi-line ANSI block-letter art.
// Returns the original text as a fallback when:
//   - the font has no character data (e.g. parse returned empty charset), or
//   - the rendered width would exceed maxWidth.
//
// The returned string may contain ANSI escape sequences and embedded newlines.
func (f *Font) Render(text string, maxWidth int) (string, error) {
	if len(text) == 0 {
		return "", nil
	}
	if len(f.chars) == 0 {
		return text, nil
	}
	if maxWidth > 0 && f.MeasureWidth(text) > maxWidth {
		return text, nil
	}

	height := f.Height()
	if height == 0 {
		return text, nil
	}

	// DOS 16-color palette mapped to hex RGB.
	dosColors := []string{
		"#000000", "#0000aa", "#00aa00", "#00aaaa",
		"#aa0000", "#aa00aa", "#aa5500", "#aaaaaa",
		"#555555", "#5555ff", "#55ff55", "#55ffff",
		"#ff5555", "#ff55ff", "#ffff55", "#ffffff",
	}
	rst := "\x1b[0m"
	sp := int(f.Spacing)
	runes := []rune(text)

	rows := make([]strings.Builder, height)

	for charIdx, ch := range runes {
		ascii := byte(ch)
		charRows := f.chars[ascii]
		cw := f.charWidth(ascii)
		if cw == 0 {
			cw = 1
		}

		for rowIdx := 0; rowIdx < height; rowIdx++ {
			var rowCells []cell
			if rowIdx < len(charRows) {
				rowCells = charRows[rowIdx]
			}

			written := 0
			for _, c := range rowCells {
				fg := c.attr & 0x0f
				bg := (c.attr >> 4) & 0x07
				var fgEsc, bgEsc string
				if int(fg) < len(dosColors) {
					r2, g2, b2 := parseHexRGB(dosColors[fg])
					fgEsc = fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r2, g2, b2)
				}
				if int(bg) < len(dosColors) {
					r2, g2, b2 := parseHexRGB(dosColors[bg])
					bgEsc = fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r2, g2, b2)
				}
				var glyph string
				if c.char == 0 {
					glyph = " "
				} else {
					glyph = cp437ToUTF8(c.char)
				}
				rows[rowIdx].WriteString(fgEsc + bgEsc + glyph + rst)
				written++
			}
			// Pad to character column width so columns align.
			for written < cw {
				rows[rowIdx].WriteByte(' ')
				written++
			}
			// Inter-character spacing (not after the last char).
			if charIdx < len(runes)-1 && sp > 0 {
				rows[rowIdx].WriteString(strings.Repeat(" ", sp))
			}
		}
	}

	var sb strings.Builder
	for i, row := range rows {
		sb.WriteString(row.String())
		if i < height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

// cp437ToUTF8 converts a CP437 byte to its UTF-8 equivalent.
// Bytes 0x01–0x06 (smiley faces, card suits) are mapped to safe single-width
// Unicode block/bullet chars to avoid double-width emoji rendering in modern
// terminals. All other bytes use their standard CP437 Unicode equivalents.
func cp437ToUTF8(b byte) string {
	table := map[byte]string{
		// Block / shade characters — primary TDF art building blocks.
		0xB0: "░", 0xB1: "▒", 0xB2: "▓", 0xDB: "█",
		0xDC: "▄", 0xDD: "▌", 0xDE: "▐", 0xDF: "▀",
		// Box-drawing.
		0xC4: "─", 0xCD: "═", 0xB3: "│", 0xBA: "║",
		0xDA: "┌", 0xBF: "┐", 0xC0: "└", 0xD9: "┘",
		0xC9: "╔", 0xBB: "╗", 0xC8: "╚", 0xBC: "╝",
		// Low-byte range 0x01–0x06: smileys and card suits appear as floating
		// decorative glyphs in some fonts — render as spaces to suppress them
		// while preserving column spacing.
		0x01: " ", 0x02: " ", 0x03: " ",
		0x04: " ", 0x05: " ", 0x06: " ",
		0x07: "·", 0x08: " ", 0x09: " ", 0x0A: " ",
		0x0B: "♂", 0x0C: "♀", 0x0E: "♪", 0x0F: "✦",
		0x10: "►", 0x11: "◄", 0x12: "↕", 0x13: "‼",
		0x14: "¶", 0x15: "§", 0x16: "▬", 0x17: "↨",
		0x18: "↑", 0x19: "↓", 0x1A: "→", 0x1B: "←",
		0x1C: "∟", 0x1D: "↔", 0x1E: "▲", 0x1F: "▼",
	}
	if s, ok := table[b]; ok {
		return s
	}
	if b >= 32 && b < 127 {
		return string(rune(b))
	}
	return "·"
}

// parseHexRGB parses "#rrggbb" into R, G, B components.
func parseHexRGB(hex string) (uint8, uint8, uint8) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	parse := func(s string) uint8 {
		v, _ := strconv.ParseUint(s, 16, 8)
		return uint8(v)
	}
	return parse(hex[0:2]), parse(hex[2:4]), parse(hex[4:6])
}
