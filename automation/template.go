package automation

import (
	"fmt"
	"strings"
)

type Template struct {
	Segments []Segment
}

type Segment interface {
	Eval(local map[string]string, globals *VarStore) (string, error)
}

type LiteralSegment struct {
	Value string
}

type VarSegment struct {
	IsGlobal bool
	Name     string
	Default  *string
}

func (l LiteralSegment) Eval(_ map[string]string, _ *VarStore) (string, error) {
	return l.Value, nil
}

func (v VarSegment) Eval(local map[string]string, globals *VarStore) (string, error) {
	var value string
	var ok bool

	if v.IsGlobal {
		value, ok = globals.Get(v.Name)
	} else {
		value, ok = local[v.Name]
	}

	if ok {
		return value, nil
	}

	if v.Default != nil {
		return *v.Default, nil
	}

	return "", fmt.Errorf("undefined variable: %s", v.Name)
}

func ParseTemplate(input string) (*Template, error) {
	var segments []Segment
	i := 0

	for i < len(input) {

		start := strings.Index(input[i:], "{{")
		if start == -1 {
			segments = append(segments, LiteralSegment{Value: input[i:]})
			break
		}

		start += i

		if start > i {
			segments = append(segments, LiteralSegment{
				Value: input[i:start],
			})
		}

		end := strings.Index(input[start:], "}}")
		if end == -1 {
			return nil, fmt.Errorf("unterminated template")
		}
		end += start

		content := strings.TrimSpace(input[start+2 : end])

		parts := strings.SplitN(content, "|", 2)
		key := parts[0]

		var defaultVal *string
		if len(parts) == 2 {
			d := parts[1]
			defaultVal = &d
		}

		isGlobal := false
		name := key

		if strings.HasPrefix(key, "vars.") {
			isGlobal = true
			name = strings.TrimPrefix(key, "vars.")
		}

		segments = append(segments, VarSegment{
			IsGlobal: isGlobal,
			Name:     name,
			Default:  defaultVal,
		})

		i = end + 2
	}

	return &Template{Segments: segments}, nil
}

func (t *Template) Render(local map[string]string, globals *VarStore) (string, error) {
	var b strings.Builder

	for _, seg := range t.Segments {
		val, err := seg.Eval(local, globals)
		if err != nil {
			return "", err
		}
		b.WriteString(val)
	}

	return b.String(), nil
}
