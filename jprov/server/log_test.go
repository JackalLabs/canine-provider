package server

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestContextHandler(t *testing.T) {
	boolKey := "someBoolKey"
	boolValue := false

	attr := slog.Bool(boolKey, boolValue)
	attrs := []slog.Attr{}
	attrs = append(attrs, attr)

	ctx := context.WithValue(context.Background(), slogFields, attrs)

	var buf strings.Builder
	textHandler := slog.NewTextHandler(
		&buf,
		&slog.HandlerOptions{},
	)
	testHandler := contextHandler{textHandler}

	rec := slog.NewRecord(time.Time{}, slog.LevelInfo, "message", 0)

	err := testHandler.Handle(ctx, rec)
	if err != nil {
		t.Fatal(err)
	}

	want := "level=INFO msg=message someBoolKey=false"
	got := buf.String()
	got = strings.TrimSuffix(got, "\n")
	if want != got {
		t.Errorf("want: %s\ngot: %s\n", want, got)
	}
}
