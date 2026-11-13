package cli

import (
	"io"
	"os"
	"sync"

	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/prompt"
	"github.com/spf13/cobra"
)

// Global CLI flags shared across commands.
type globalFlags struct {
	Debug  bool
	Format output.Format
	Quiet  bool
}

func (g *globalFlags) console() *output.Console {
	return output.NewConsole(os.Stdout, os.Stderr, g.Format, g.Quiet)
}

func bindGlobalFlags(cmd *cobra.Command, g *globalFlags) {
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "Afficher les détails internes en cas d'erreur")
	format := string(output.FormatHuman)
	cmd.PersistentFlags().StringVar((*string)(&g.Format), "format", format, "Format de sortie : human, json")
	cmd.PersistentFlags().BoolVar(&g.Quiet, "quiet", false, "Réduire la sortie (sauf erreurs et JSON)")
}

var stderr io.Writer = os.Stderr

var (
	sharedPrompter     *prompt.Prompter
	sharedPrompterOnce sync.Once
)

// stdinPrompter returns the single prompter reading os.Stdin. Each prompter
// buffers its input, so a new one per question would lose the answers already
// buffered by the previous one when stdin is a pipe or a file.
func stdinPrompter() *prompt.Prompter {
	sharedPrompterOnce.Do(func() {
		sharedPrompter = prompt.New(os.Stdin, os.Stdout)
	})
	return sharedPrompter
}
