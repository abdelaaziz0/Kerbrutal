package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/abdelaaziz0/kerbrutal/mutate"
	"github.com/abdelaaziz0/kerbrutal/util"
	"github.com/spf13/cobra"
)

var mutateCmd = &cobra.Command{
	Use:   "mutate",
	Short: "Generate AD username permutations from employee names",
	Long: `Takes a file of human names (one per line) and generates all
likely Active Directory username formats. Output can be piped directly
into userenum or saved to a file.

Supported input formats:
  "John Doe", "Doe, John", "Dr. John M. Doe III", "Maria Garcia"

Mutation levels:
  standard  ~8 formats per name  (most common patterns)
  extended  ~15 formats          (includes underscores, reversed)
  full      ~22 formats          (includes middle initials, hyphen splits)`,
	RunE: runMutate,
}

var (
	namesFile   string
	mutateLevel string
	mutateOut   string
)

func init() {
	rootCmd.AddCommand(mutateCmd)
	mutateCmd.Flags().StringVarP(&namesFile, "names", "n", "", "File containing employee names (one per line)")
	mutateCmd.Flags().StringVar(&mutateLevel, "level", "standard", "Mutation depth: standard, extended, or full")
	mutateCmd.Flags().StringVarP(&mutateOut, "output", "o", "", "Output file for generated usernames (default: stdout)")
	mutateCmd.MarkFlagRequired("names")
}

func runMutate(cmd *cobra.Command, args []string) error {
	if logger.Log == nil {
		logger = util.NewLogger(verbose, logFileName, false)
	}
	level := mutate.LevelStandard
	switch strings.ToLower(mutateLevel) {
	case "extended":
		level = mutate.LevelExtended
	case "full":
		level = mutate.LevelFull
	}

	usernames, err := mutate.GenerateFromFile(namesFile, level, logger)
	if err != nil {
		return err
	}

	logger.Log.Infof("Mutation complete: Generated %d usernames from %s (level: %s)", len(usernames), namesFile, mutateLevel)

	if mutateOut != "" {
		data := strings.Join(usernames, "\n") + "\n"
		return os.WriteFile(mutateOut, []byte(data), 0644)
	}

	for _, u := range usernames {
		fmt.Println(u)
	}
	return nil
}
