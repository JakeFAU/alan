// Command consumer observes an exact, already revoked dispatch authority.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/JakeFAU/alan/runtime"
)

func observe(ctx context.Context, args []string, out io.Writer) (err error) {
	flags := flag.NewFlagSet("revocation-consumer", flag.ContinueOnError)
	path := flags.String("db", "", "retained store")
	namespace := flags.String("namespace", "", "original store namespace")
	subject := flags.String("subject", "", "original run subject")
	generation := flags.Int64("generation", 0, "original run generation")
	scope := flags.Bool("scope", false, "observe the run's owning scope")
	timeout := flags.Duration("timeout", 5*time.Second, "maximum wait")
	if err = flags.Parse(args); err != nil {
		return err
	}
	if *path == "" || *namespace == "" || *subject == "" || *generation <= 0 || *timeout <= 0 || flags.NArg() != 0 {
		return fmt.Errorf("require -db, -namespace, -subject, positive -generation and -timeout")
	}
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	s, err := runtime.Open(ctx, *path, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Close()) }()
	target := runtime.DispatchAuthority{Namespace: *namespace, Run: runtime.RunID{Subject: *subject, Generation: *generation}, OwningScope: *scope}
	if err = s.AwaitDispatchQuiescence(ctx, target); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Target         runtime.DispatchAuthority `json:"target"`
		LocalQuiescent bool                      `json:"local_quiescent"`
	}{target, true})
}

func main() {
	if err := observe(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
