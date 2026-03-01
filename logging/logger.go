package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

type PrettyOptions struct {
	Level slog.Leveler
	Color bool
}

type PrettyHandler struct {
	opts   PrettyOptions
	attrs  []slog.Attr
	groups []string
	w      io.Writer
	mu     *sync.Mutex
}

func New(level string) *slog.Logger {
	handler := &PrettyHandler{
		opts: PrettyOptions{
			Level: parseLevel(level),
			Color: shouldColor(os.Stderr),
		},
		w:  os.Stderr,
		mu: &sync.Mutex{},
	}

	return slog.New(handler)
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}

	return level >= minLevel
}

func (h *PrettyHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	var b strings.Builder
	timestamp := record.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	b.WriteString(h.formatTimestamp(timestamp))
	b.WriteByte(' ')
	b.WriteString(h.formatLevel(record.Level))
	b.WriteByte(' ')
	b.WriteString(record.Message)

	for _, attr := range attrs {
		h.appendAttr(&b, "", attr)
	}

	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h.clone()
	next.attrs = append(next.attrs, attrs...)
	return next
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

func (h *PrettyHandler) clone() *PrettyHandler {
	attrs := append([]slog.Attr(nil), h.attrs...)
	groups := append([]string(nil), h.groups...)

	return &PrettyHandler{
		opts:   h.opts,
		attrs:  attrs,
		groups: groups,
		w:      h.w,
		mu:     h.mu,
	}
}

func (h *PrettyHandler) appendAttr(b *strings.Builder, prefix string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	groupPrefix := prefix
	if groupPrefix == "" && len(h.groups) > 0 {
		groupPrefix = strings.Join(h.groups, ".")
	}

	key := joinKey(groupPrefix, attr.Key)

	switch attr.Value.Kind() {
	case slog.KindGroup:
		nextPrefix := key
		if nextPrefix == "" {
			nextPrefix = groupPrefix
		}
		for _, nested := range attr.Value.Group() {
			h.appendAttr(b, nextPrefix, nested)
		}
	default:
		if key == "" {
			return
		}
		b.WriteByte(' ')
		if h.opts.Color {
			b.WriteString(colorBlue)
			b.WriteString(key)
			b.WriteString(colorReset)
		} else {
			b.WriteString(key)
		}
		b.WriteByte('=')
		b.WriteString(formatValue(attr.Value.Any()))
	}
}

func joinKey(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}

	return strings.Join(filtered, ".")
}

func (h *PrettyHandler) formatTimestamp(t time.Time) string {
	value := t.Format("2006-01-02 15:04:05")
	if !h.opts.Color {
		return value
	}

	return colorDim + value + colorReset
}

func (h *PrettyHandler) formatLevel(level slog.Level) string {
	label := strings.ToUpper(level.String())
	color := ""

	switch {
	case level < slog.LevelInfo:
		color = colorCyan
	case level < slog.LevelWarn:
		color = colorGreen
	case level < slog.LevelError:
		color = colorYellow
	default:
		color = colorRed
	}

	if !h.opts.Color {
		return label
	}

	return color + label + colorReset
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func shouldColor(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0 && os.Getenv("NO_COLOR") == ""
}

func formatValue(value any) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case fmt.Stringer:
		return strconv.Quote(v.String())
	case error:
		return strconv.Quote(v.Error())
	case time.Time:
		return strconv.Quote(v.Format(time.RFC3339))
	default:
		return fmt.Sprintf("%v", value)
	}
}
