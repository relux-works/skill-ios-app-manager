package cli

import (
	"fmt"
	"os"

	cleaner "github.com/relux-works/ios-app-manager/internal/clean"
	"github.com/spf13/cobra"
)

type cleanManager interface {
	QuickClean(projectRoot string) (cleaner.Result, error)
	DeepClean(projectRoot string) (cleaner.Result, error)
	KillXcode() error
}

var (
	cleanManagerFactory = func() cleanManager {
		return cleaner.NewManager()
	}
	cleanGetwd = os.Getwd
)

func newCleanCommand(_ *RootOptions) *cobra.Command {
	var deep bool
	var killXcode bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean generated artifacts and caches",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectRoot, err := cleanGetwd()
			if err != nil {
				return fmt.Errorf("resolve project root: %w", err)
			}

			manager := cleanManagerFactory()

			runDeepClean := deep || killXcode
			if killXcode {
				if err := manager.KillXcode(); err != nil {
					if _, writeErr := fmt.Fprintf(
						cmd.ErrOrStderr(),
						"warning: unable to kill Xcode before clean: %v\n",
						err,
					); writeErr != nil {
						return writeErr
					}
				}
			}

			var result cleaner.Result
			if runDeepClean {
				result, err = manager.DeepClean(projectRoot)
				if err != nil {
					return fmt.Errorf("deep clean: %w", err)
				}
			} else {
				result, err = manager.QuickClean(projectRoot)
				if err != nil {
					return fmt.Errorf("quick clean: %w", err)
				}
			}

			if runDeepClean {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "deep clean completed"); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "quick clean completed"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "cleaned paths:"); err != nil {
				return err
			}
			for _, path := range result.CleanedPaths {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", path); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"estimated freed space: %s\n",
				cleaner.FormatBytes(result.FreedBytes),
			)
			return err
		},
	}

	cmd.Flags().BoolVar(&deep, "deep", false, "Run deep clean (global caches + local artifacts)")
	cmd.Flags().BoolVar(
		&killXcode,
		"kill-xcode",
		false,
		"Kill Xcode before cleaning (implies --deep)",
	)

	return cmd
}
