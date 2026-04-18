package exec

import (
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
	"path/filepath"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	traceotel "go.opentelemetry.io/otel/trace"
)

var (
	ErrDot       = osexec.ErrDot
	ErrNotFound  = osexec.ErrNotFound
	ErrWaitDelay = osexec.ErrWaitDelay
)

type Error = osexec.Error
type ExitError = osexec.ExitError

var tracer = otel.Tracer("kyoo.transcoder.cli")

type Cmd struct {
	*osexec.Cmd
	span     traceotel.Span
	spanCtx  context.Context
	spanned  bool
	spanDone bool
}

func LookPath(file string) (string, error) {
	return osexec.LookPath(file)
}

func Command(name string, arg ...string) *Cmd {
	return wrap(context.Background(), osexec.Command(name, arg...))
}

func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return wrap(ctx, osexec.CommandContext(ctx, name, arg...))
}

func wrap(ctx context.Context, cmd *osexec.Cmd) *Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Cmd{Cmd: cmd, spanCtx: ctx}
}

func (c *Cmd) Run() error {
	c.startSpan()
	err := c.Cmd.Run()
	c.endSpan(err)
	return err
}

func (c *Cmd) Output() ([]byte, error) {
	c.startSpan()
	output, err := c.Cmd.Output()
	c.endSpan(err)
	return output, err
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	c.startSpan()
	output, err := c.Cmd.CombinedOutput()
	c.endSpan(err)
	return output, err
}

func (c *Cmd) Start() error {
	c.startSpan()
	err := c.Cmd.Start()
	if err != nil {
		c.endSpan(err)
	}
	return err
}

func (c *Cmd) Wait() error {
	err := c.Cmd.Wait()
	c.endSpan(err)
	return err
}

func (c *Cmd) startSpan() {
	if c == nil || c.spanned {
		return
	}

	ctx := c.spanCtx
	if ctx == nil {
		ctx = context.Background()
	}

	attrs := []attribute.KeyValue{
		attribute.String("process.command", c.Path),
	}
	if len(c.Args) > 1 {
		attrs = append(attrs, attribute.StringSlice("process.command_args", c.Args[1:]))
	}
	if c.Dir != "" {
		attrs = append(attrs, attribute.String("process.working_directory", c.Dir))
	}

	_, span := tracer.Start(
		ctx,
		fmt.Sprintf("exec %s", filepath.Base(c.Path)),
		traceotel.WithAttributes(attrs...),
	)
	c.span = span
	c.spanned = true
	c.spanDone = false
}

func (c *Cmd) endSpan(err error) {
	if c == nil || !c.spanned || c.spanDone || c.span == nil {
		return
	}

	if err == nil {
		c.span.SetAttributes(attribute.Int("process.exit.code", 0))
		c.span.SetStatus(codes.Ok, "")
		c.span.End()
		c.spanDone = true
		return
	}

	c.span.RecordError(err)

	if exitErr, ok := errors.AsType[*osexec.ExitError](err); ok {
		c.span.SetAttributes(attribute.Int("process.exit.code", exitErr.ExitCode()))
	}

	c.span.SetStatus(codes.Error, err.Error())
	c.span.End()
	c.spanDone = true
}
