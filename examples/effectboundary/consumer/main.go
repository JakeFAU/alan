// Command consumer uses only ALAN public packages. Its reports are host output;
// none are fed back into the authored parent's ordinary observation stream.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/runtime"
	"io"
	"os"
)

func execute(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected init, run, recover, inspect or replay")
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("db", "", "retained store")
	endpoint := flags.String("endpoint", "", "literal-loopback endpoint")
	mode := flags.String("mode", "success", "success, selected-error, unknown")
	parent := flags.Bool("parent", true, "ordinary invocation parent")
	peer := flags.Bool("peer", true, "independently enabled delta-two peer")
	fail := flags.Bool("private-failure", false, "child fails privately after Request")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *path == "" || flags.NArg() != 0 {
		return fmt.Errorf("-db required and no positional arguments accepted")
	}
	encode := func(v any) error { return json.NewEncoder(out).Encode(v) }
	switch command {
	case "init":
		d, err := effectboundary.New(effectboundary.Config{Parent: *parent, Peer: *peer, FailAfterRequest: *fail, Delta: 1})
		if err != nil {
			return err
		}
		s, err := runtime.Create(ctx, *path, d.Provision, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		if err = s.Close(); err != nil {
			return err
		}
		return encode(map[string]any{"source": d.Source.ID(), "child": d.Child.ID(), "parent": d.Parent.ID()})
	case "run", "recover":
		s, err := runtime.Open(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		defer s.Close()
		report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: effectboundary.Registry(*endpoint, *mode)})
		if err != nil {
			return err
		}
		return encode(report)
	case "inspect":
		v, err := runtime.Inspect(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(v)
	case "replay":
		v, err := runtime.Replay(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(v)
	default:
		return fmt.Errorf("unsupported command %q", command)
	}
}
func main() {
	if err := execute(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
