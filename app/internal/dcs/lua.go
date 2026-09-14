package dcs

import (
	"fmt"
	"strconv"
	"strings"
)

// kv is one ordered entry of a Lua table; key is string or int.
type kv struct {
	key any
	val any
}

// T is an ordered Lua table (DCS mission files are Lua table literals).
type T []kv

func k(key any, val any) kv { return kv{key, val} }

func tbl(pairs ...kv) T { return T(pairs) }

// arr builds a 1-based Lua array table.
func arr(vals ...any) T {
	t := make(T, 0, len(vals))
	for i, v := range vals {
		t = append(t, kv{i + 1, v})
	}
	return t
}

// point returns the common {x, y} pair.
func point(x, y float64) T { return tbl(k("x", x), k("y", y)) }

func luaString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\r':
		case '\n':
			// Lua line continuation, as written by the DCS mission editor.
			b.WriteString("\\\n")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func luaKey(key any) string {
	switch t := key.(type) {
	case int:
		return "[" + strconv.Itoa(t) + "]"
	case string:
		return "[" + luaString(t) + "]"
	}
	return fmt.Sprintf("[%v]", key)
}

func writeValue(b *strings.Builder, v any, indent int) {
	switch t := v.(type) {
	case T:
		if len(t) == 0 {
			b.WriteString("{}")
			return
		}
		pad := strings.Repeat("\t", indent)
		b.WriteString("\n" + pad + "{\n")
		for _, e := range t {
			b.WriteString(pad + "\t" + luaKey(e.key) + " = ")
			writeValue(b, e.val, indent+1)
			b.WriteString(",\n")
		}
		b.WriteString(pad + "}")
	case string:
		b.WriteString(luaString(t))
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.Itoa(t))
	case int64:
		b.WriteString(strconv.FormatInt(t, 10))
	case float64:
		b.WriteString(strconv.FormatFloat(t, 'f', -1, 64))
	case nil:
		b.WriteString("nil")
	default:
		b.WriteString(fmt.Sprintf("%v", t))
	}
}

// Serialize renders "name = { ... }" like the DCS mission editor does.
func Serialize(name string, t T) string {
	var b strings.Builder
	b.WriteString(name + " = ")
	writeValue(&b, t, 0)
	b.WriteString(" -- end of " + name + "\n")
	return b.String()
}
