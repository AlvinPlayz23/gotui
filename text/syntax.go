package text

import (
	"strings"
	"unicode"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
)

// SyntaxToken names a lexical class used by the built-in lightweight highlighter.
type SyntaxToken string

const (
	TokenPlain       SyntaxToken = "plain"
	TokenKeyword     SyntaxToken = "keyword"
	TokenString      SyntaxToken = "string"
	TokenNumber      SyntaxToken = "number"
	TokenComment     SyntaxToken = "comment"
	TokenType        SyntaxToken = "type"
	TokenFunction    SyntaxToken = "function"
	TokenOperator    SyntaxToken = "operator"
	TokenPunctuation SyntaxToken = "punctuation"
)

// TokenStyle is the foreground, background, and attributes for one syntax class.
type TokenStyle struct {
	FG   color.RGBA
	BG   color.RGBA
	Attr cell.Attribute
}

// SyntaxStyle maps lexical classes to styles. Zero values inherit the plain style.
type SyntaxStyle struct {
	Plain       TokenStyle
	Keyword     TokenStyle
	String      TokenStyle
	Number      TokenStyle
	Comment     TokenStyle
	Type        TokenStyle
	Function    TokenStyle
	Operator    TokenStyle
	Punctuation TokenStyle
}

// DefaultSyntaxStyle returns a restrained terminal-friendly palette.
func DefaultSyntaxStyle() SyntaxStyle {
	return SyntaxStyle{
		Plain:       TokenStyle{FG: color.FromInts(220, 220, 220, 255)},
		Keyword:     TokenStyle{FG: color.FromInts(198, 120, 221, 255), Attr: cell.AttrBold},
		String:      TokenStyle{FG: color.FromInts(152, 195, 121, 255)},
		Number:      TokenStyle{FG: color.FromInts(209, 154, 102, 255)},
		Comment:     TokenStyle{FG: color.FromInts(128, 136, 148, 255), Attr: cell.AttrItalic},
		Type:        TokenStyle{FG: color.FromInts(229, 192, 123, 255)},
		Function:    TokenStyle{FG: color.FromInts(97, 175, 239, 255)},
		Operator:    TokenStyle{FG: color.FromInts(171, 178, 191, 255)},
		Punctuation: TokenStyle{FG: color.FromInts(171, 178, 191, 255)},
	}
}

func (s SyntaxStyle) style(token SyntaxToken) TokenStyle {
	var st TokenStyle
	switch token {
	case TokenKeyword:
		st = s.Keyword
	case TokenString:
		st = s.String
	case TokenNumber:
		st = s.Number
	case TokenComment:
		st = s.Comment
	case TokenType:
		st = s.Type
	case TokenFunction:
		st = s.Function
	case TokenOperator:
		st = s.Operator
	case TokenPunctuation:
		st = s.Punctuation
	default:
		st = s.Plain
	}
	if st.FG == (color.RGBA{}) {
		st.FG = s.Plain.FG
	}
	if st.BG == (color.RGBA{}) {
		st.BG = s.Plain.BG
	}
	st.Attr |= s.Plain.Attr
	return st
}

// HighlightCode applies a dependency-free lexical highlighter for common languages.
// It intentionally does not parse language grammar; unknown constructs stay plain.
func HighlightCode(source, language string, style SyntaxStyle) StyledText {
	if style == (SyntaxStyle{}) {
		style = DefaultSyntaxStyle()
	}
	lang := strings.ToLower(strings.TrimSpace(language))
	keywords := keywordSet(lang)
	runes := []rune(source)
	chunks := make([]Chunk, 0, len(runes)/8+1)
	appendToken := func(token SyntaxToken, text string) {
		if text == "" {
			return
		}
		st := style.style(token)
		if n := len(chunks); n > 0 && chunks[n-1].FG == st.FG && chunks[n-1].BG == st.BG && chunks[n-1].Attr == st.Attr {
			chunks[n-1].Text += text
		} else {
			chunks = append(chunks, Chunk{Text: text, FG: st.FG, BG: st.BG, Attr: st.Attr})
		}
	}
	for i := 0; i < len(runes); {
		start := i
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
			appendToken(TokenPlain, string(runes[start:i]))
			continue
		}
		if isCommentStart(runes, i, lang) {
			if r == '/' && i+1 < len(runes) && runes[i+1] == '*' {
				i += 2
				for i < len(runes) {
					if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
			} else {
				for i < len(runes) && runes[i] != '\n' {
					i++
				}
			}
			appendToken(TokenComment, string(runes[start:i]))
			continue
		}
		if r == '\'' || r == '"' || r == '`' {
			quote := r
			i++
			for i < len(runes) {
				if runes[i] == '\\' && quote != '`' && i+1 < len(runes) {
					i += 2
					continue
				}
				if runes[i] == quote {
					i++
					break
				}
				i++
			}
			appendToken(TokenString, string(runes[start:i]))
			continue
		}
		if unicode.IsDigit(r) {
			i++
			for i < len(runes) && (unicode.IsDigit(runes[i]) || unicode.IsLetter(runes[i]) || strings.ContainsRune("._", runes[i])) {
				i++
			}
			appendToken(TokenNumber, string(runes[start:i]))
			continue
		}
		if unicode.IsLetter(r) || r == '_' || r == '$' {
			i++
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_' || runes[i] == '$') {
				i++
			}
			word := string(runes[start:i])
			token := TokenPlain
			if keywords[word] {
				token = TokenKeyword
			} else if nextNonSpace(runes, i) == '(' {
				token = TokenFunction
			} else if startsUpper(word) {
				token = TokenType
			}
			appendToken(token, word)
			continue
		}
		i++
		if strings.ContainsRune("+-*/%=!<>|&^~?:", r) {
			appendToken(TokenOperator, string(r))
		} else if strings.ContainsRune("()[]{}.,;", r) {
			appendToken(TokenPunctuation, string(r))
		} else {
			appendToken(TokenPlain, string(r))
		}
	}
	return StyledText{Chunks: chunks}
}

func startsUpper(s string) bool {
	for _, r := range s {
		return unicode.IsUpper(r)
	}
	return false
}

func nextNonSpace(r []rune, i int) rune {
	for i < len(r) && unicode.IsSpace(r[i]) {
		i++
	}
	if i < len(r) {
		return r[i]
	}
	return 0
}

func isCommentStart(r []rune, i int, lang string) bool {
	if i >= len(r) {
		return false
	}
	if r[i] == '#' && (lang == "python" || lang == "py" || lang == "shell" || lang == "sh" || lang == "bash" || lang == "yaml" || lang == "yml" || lang == "toml" || lang == "") {
		return true
	}
	return r[i] == '/' && i+1 < len(r) && (r[i+1] == '/' || r[i+1] == '*')
}

func keywordSet(lang string) map[string]bool {
	common := "break case catch class const continue default defer do else export extends finally for func function if import in interface let new package private protected public return static struct switch throw try type var while yield async await true false null nil this self super"
	switch lang {
	case "go", "golang":
		common = "break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var bool byte complex64 complex128 error float32 float64 int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr true false iota nil"
	case "python", "py":
		common = "and as assert async await break class continue def del elif else except False finally for from global if import in is lambda None nonlocal not or pass raise return True try while with yield self"
	case "json":
		common = "true false null"
	case "sql":
		common = "select from where insert into values update set delete create table alter join left right inner outer on as and or not null is group by order having limit offset distinct primary key foreign references"
	}
	out := make(map[string]bool)
	for _, word := range strings.Fields(common) {
		out[word] = true
	}
	return out
}
