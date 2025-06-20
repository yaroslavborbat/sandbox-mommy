package attach

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yaroslavborbat/sandbox-mommy/api/client/kubeclient"
	subv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/subresources/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/clientconfig"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/template"
)

const (
	example = `  # Attach to the sandbox 'my-sandbox':
  {{ProgramName}} attach my-sandbox
  {{ProgramName}} attach my-sandbox -n my-namespace`

	long = `Attach to a sandbox.

The sandbox must be in the running phase.`
)

type attach struct{}

func NewAttachSandboxCommand() *cobra.Command {
	a := &attach{}

	cmd := &cobra.Command{
		Use:     "attach",
		Short:   "Attach to a sandbox",
		Example: example,
		Long:    long,
		Args:    cobra.ExactArgs(1),
		RunE:    a.Run,
	}

	cmd.SetUsageTemplate(template.UsageTemplate())
	return cmd
}

func (a *attach) Run(cmd *cobra.Command, args []string) error {
	name := args[0]
	client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	interrupt := make(chan os.Signal, 1)
	go func() {
		<-interrupt
		close(interrupt)
	}()
	signal.Notify(interrupt, os.Interrupt)

	for {
		err := connect(name, namespace, client)
		if err == nil {
			continue
		}
		if errors.Is(err, ErrorInterrupt) || strings.Contains(err.Error(), "not found") {
			return ignoreInterrupt(err)
		}

		var e *websocket.CloseError
		if errors.As(err, &e) {
			switch e.Code {
			case websocket.CloseGoingAway:
				cmd.PrintErrln("\nYou were disconnected from the console.")
				return nil
			case websocket.CloseAbnormalClosure:
				cmd.PrintErrln("\nYou were disconnected from the console.")
			}
		} else {
			cmd.PrintErrf("%v\n", err)
		}

		select {
		case <-interrupt:
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}

func connect(name string, namespace string, client kubeclient.Client,
) error {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// in -> stdinWriter | stdinReader -> attach
	// out <- stdoutReader | stdoutWriter <- attach
	resChan := make(chan error)
	runningChan := make(chan error)

	go func() {
		con, err := client.Sandboxes(namespace).Attach(name, &subv1alpha1.Attach{ConnectionTimeout: metav1.Duration{Duration: 1 * time.Minute}})
		runningChan <- err

		if err != nil {
			return
		}

		resChan <- con.Stream(kubeclient.StreamOptions{
			In:  stdinReader,
			Out: stdoutWriter,
		})
	}()

	err := <-runningChan
	if err != nil {
		return err
	}

	return attachTerm(stdoutReader, stdinWriter, name, resChan)
}

func ignoreInterrupt(err error) error {
	if errors.Is(err, ErrorInterrupt) {
		return nil
	}
	return err
}

var ErrorInterrupt = errors.New("interrupt")

func attachTerm(stdoutReader *io.PipeReader, stdinWriter *io.PipeWriter, name string, resChan <-chan error) (err error) {
	writeStop := make(chan error)
	readStop := make(chan error)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("make raw terminal failed: %w", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), state)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully connected to %s console. The escape sequence is ^]\n", name)

	out := os.Stdout
	go func() {
		defer close(readStop)
		_, err := io.Copy(out, stdoutReader)
		readStop <- err
	}()

	stdinCh := make(chan []byte)
	go func() {
		in := os.Stdin
		defer close(stdinCh)
		buf := make([]byte, 1024)
		for {
			// reading from stdin
			n, err := in.Read(buf)
			if err != nil && err != io.EOF {
				return
			}
			if n == 0 && err == io.EOF {
				return
			}

			// the escape sequence
			if buf[0] == 29 {
				return
			}

			stdinCh <- buf[0:n]
		}
	}()

	go func() {
		defer close(writeStop)

		_, err := stdinWriter.Write([]byte("\r"))
		if err == io.EOF {
			return
		}

		for b := range stdinCh {
			_, err = stdinWriter.Write(b)
			if err == io.EOF {
				return
			}
		}
	}()

	select {
	case <-writeStop:
		return ErrorInterrupt
	case <-readStop:
		return ErrorInterrupt
	case err = <-resChan:
		return err
	}
}
