package provider

import (
	"context"
	"log/slog"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Translates logs to tflog. This way output from the megaportgo library will also be
// piped into terraform with the correct level.
//
// NOTE: logs without context will not work because tflog extracts the underlying logger
// from the context, but this shouldn't be an issue as context is enforced in the megaportgo
// linting rules.
type tfhandler struct{}

func (tfhandler) Handle(ctx context.Context, r slog.Record) error {
	addl := make(map[string]interface{})
	r.Attrs(func(s slog.Attr) bool {
		addl[s.Key] = s.Value.String()
		return true
	})
	switch r.Level {
	case slog.LevelDebug:
		tflog.Debug(ctx, r.Message, addl)
	case slog.LevelInfo:
		tflog.Info(ctx, r.Message, addl)
	case slog.LevelWarn:
		tflog.Warn(ctx, r.Message, addl)
	case slog.LevelError:
		tflog.Error(ctx, r.Message, addl)
	default:
		tflog.Info(ctx, r.Message, addl)
	}
	return nil
}

func (tfhandler) Enabled(context.Context, slog.Level) bool { return true }
func (tfhandler) WithAttrs(attrs []slog.Attr) slog.Handler { panic("unimplemented") }
func (tfhandler) WithGroup(name string) slog.Handler       { panic("unimplemented") }
