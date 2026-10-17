package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Level      string // debug, info, warn, error
	Format     string // local (readability-focused), cloud (optimized for parsing by CloudWatch etc.)
	Env        string // environment name (logged only in cloud format)
	AppVersion string // app version (logged only in cloud format)
}

// localHandler outputs logs as space-separated values without key names for
// built-in fields (time, level, source, msg). Custom attributes use key=value.
type localHandler struct {
	mu       sync.Mutex
	out      io.Writer
	level    slog.Level
	wd       string // working directory for relative source paths
	preAttrs []slog.Attr
	groups   []string
}

func newLocalHandler(out io.Writer, level slog.Level) *localHandler {
	wd, _ := os.Getwd()
	return &localHandler{out: out, level: level, wd: wd}
}

func (h *localHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle output log with simple format
// e.g. "2026-04-26T14:47:49.654+09:00 INFO cmd/sample/main.go:23 hello"
func (h *localHandler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	if !r.Time.IsZero() {
		buf.WriteString(r.Time.In(time.Local).Format("2006-01-02T15:04:05.000Z07:00"))
		buf.WriteByte(' ')
	}

	buf.WriteString(r.Level.String())

	if r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := frames.Next()
		src := f.File
		if h.wd != "" {
			if rel, err := filepath.Rel(h.wd, f.File); err == nil {
				src = rel
			}
		}
		fmt.Fprintf(&buf, " %s:%d", src, f.Line)
	}

	buf.WriteByte(' ')
	buf.WriteString(r.Message)

	for _, a := range h.preAttrs {
		h.appendAttr(&buf, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&buf, a)
		return true
	})

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *localHandler) appendAttr(buf *bytes.Buffer, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		for _, ga := range a.Value.Group() {
			if a.Key != "" {
				ga.Key = a.Key + "." + ga.Key
			}
			h.appendAttr(buf, ga)
		}
		return
	}
	key := a.Key
	if len(h.groups) > 0 {
		key = strings.Join(h.groups, ".") + "." + key
	}
	val := a.Value.String()
	if strings.ContainsAny(val, " \t=") {
		fmt.Fprintf(buf, " %s=%q", key, val)
	} else {
		fmt.Fprintf(buf, " %s=%s", key, val)
	}
}

func (h *localHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.preAttrs)+len(attrs))
	copy(newAttrs, h.preAttrs)
	copy(newAttrs[len(h.preAttrs):], attrs)
	return &localHandler{
		out:      h.out,
		level:    h.level,
		wd:       h.wd,
		preAttrs: newAttrs,
		groups:   h.groups,
	}
}

func (h *localHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &localHandler{
		out:      h.out,
		level:    h.level,
		wd:       h.wd,
		preAttrs: h.preAttrs,
		groups:   newGroups,
	}
}

type customHandler struct {
	inner slog.Handler
}

func (h *customHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		if frames := captureStackFrames(); len(frames) > 0 {
			r.AddAttrs(slog.Any("stacktrace", frames))
		}
	}
	return h.inner.Handle(ctx, r)
}

func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &customHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *customHandler) WithGroup(name string) slog.Handler {
	return &customHandler{inner: h.inner.WithGroup(name)}
}

// stackFrame is stack trace information for 1 frame
type stackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// captureStackFrames get caller stack as array of structured frame.
// Remove frame of Slog and this package.
// Only leave frame of main app.
func captureStackFrames() []stackFrame {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(0, pcs[:])
	if n == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:n])

	result := make([]stackFrame, 0, n)
	for {
		frame, more := frames.Next()
		if !shouldSkipFrame(frame) {
			result = append(result, stackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})
		}
		if !more {
			break
		}
	}
	return result
}

func shouldSkipFrame(f runtime.Frame) bool {
	return strings.HasPrefix(f.Function, "runtime.") ||
		strings.HasPrefix(f.Function, "log/slog.") ||
		strings.Contains(f.File, "/pkg/logger/")
}

func cloudTimeReplacer(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.String(slog.TimeKey, a.Value.Time().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
	}
	return a
}

func NewLogger(cfg Config) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid LOG_LEVEL %q, fallback to 'info'\n", cfg.Level)
		level = slog.LevelInfo
	}

	var inner slog.Handler
	switch cfg.Format {
	case "local":
		// local: prioritize readability
		// (unstructured log, local timezone)
		inner = newLocalHandler(os.Stderr, level)
	case "cloud":
		// cloud: prioritize parseability for CloudWatch etc.
		// (structured log, UTC)
		opts := &slog.HandlerOptions{Level: level, AddSource: true, ReplaceAttr: cloudTimeReplacer}
		inner = slog.NewJSONHandler(os.Stderr, opts)
	default:
		// fall back to cloud format if LOG_FORMAT is invalid
		fmt.Fprintf(os.Stderr, "invalid LOG_FORMAT %q, fallback to 'cloud'\n", cfg.Format)
		opts := &slog.HandlerOptions{Level: level, AddSource: true, ReplaceAttr: cloudTimeReplacer}
		inner = slog.NewJSONHandler(os.Stderr, opts)
	}

	// custom handler that attaches a stack trace to Error-level records
	handler := &customHandler{inner: inner}
	logger := slog.New(handler)

	// add standard fields for cloud format
	if cfg.Format == "cloud" {
		logger = logger.With(
			slog.String("env", cfg.Env),
			slog.String("ver", cfg.AppVersion),
		)
	}

	return logger
}
