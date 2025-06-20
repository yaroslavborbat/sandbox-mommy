package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Output string

const (
	Stdout  Output = "stdout"
	Stderr  Output = "stderr"
	Discard Output = "discard"
)

func (o Output) String() string {
	return string(o)
}

type Format string

const (
	Json Format = "json"
	Text Format = "text"
)

func (f Format) String() string {
	return string(f)
}

func SetupSlogLogger(output Output, logFormat Format, level string) error {
	var w io.Writer

	switch output {
	case Stdout:
		w = os.Stdout
	case Stderr:
		w = os.Stderr
	case Discard:
		log := slog.New(slog.DiscardHandler)
		SetDefaultLogger(log)
		return nil
	default:
		return fmt.Errorf("unknown LogOutput: %s", output)
	}

	var detectLevel slog.Level

	switch strings.ToLower(level) {
	case "error":
		detectLevel = slog.LevelError
	case "warn":
		detectLevel = slog.LevelWarn
	case "info":
		detectLevel = slog.LevelInfo
	case "debug":
		detectLevel = slog.LevelDebug
	default:
		detectLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: detectLevel,
	}

	var log *slog.Logger
	switch logFormat {
	case Json:
		h := slog.NewJSONHandler(w, opts)
		log = slog.New(h)
	case Text, "":
		h := slog.NewTextHandler(w, opts)
		log = slog.New(h)
	default:
		return fmt.Errorf("unknown LogFormat: %s", logFormat)
	}

	SetDefaultLogger(log)
	return nil
}

func SetDefaultLogger(l *slog.Logger) {
	slog.SetDefault(slog.New(l.Handler()))
	slog.SetDefault(l)
	fromSlog := logr.FromSlogHandler(l.Handler())
	logf.SetLogger(fromSlog)
	klog.SetLogger(fromSlog)
}
