package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"strings"

	"github.com/abdelaaziz0/kerbrutal/mutate"
	"github.com/abdelaaziz0/kerbrutal/util"
	"github.com/spf13/cobra"
)

var userEnumCommand = &cobra.Command{
	Use:   "userenum [flags] <username_wordlist>",
	Short: "Enumerate valid domain usernames via Kerberos",
	Long: `Will enumerate valid usernames from a list by constructing AS-REQs to requesting a TGT from the KDC.
If no domain controller is specified, the tool will attempt to look one up via DNS SRV records.
A full domain is required. This domain will be capitalized and used as the Kerberos realm when attempting the bruteforce.
Valid usernames will be displayed on stdout. If using --mutate, pass --names instead of a wordlist.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if mutateFlag {
			if namesFileFlag == "" {
				return fmt.Errorf("--mutate requires --names <file>")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 argument (a wordlist file)")
		}
		return nil
	},
	PreRun: setupSession,
	RunE:   userEnum,
}

var (
	mutateFlag       bool
	namesFileFlag    string
	mutateLevelFlag  string
)

func init() {
	rootCmd.AddCommand(userEnumCommand)
	userEnumCommand.Flags().BoolVar(&mutateFlag, "mutate", false, "Enable name-to-username mutation (use with --names)")
	userEnumCommand.Flags().StringVar(&namesFileFlag, "names", "", "File of employee names to mutate into usernames")
	userEnumCommand.Flags().StringVar(&mutateLevelFlag, "mutate-level", "standard", "Mutation depth: standard, extended, full")
}

func userEnum(cmd *cobra.Command, args []string) error {
	var usernamelist string
	if len(args) > 0 {
		usernamelist = args[0]
	}
	usersChan := make(chan string, threads)
	defer cancel()
	defer closeValidUsersFile()
	defer closeResumeFile()

	var wg sync.WaitGroup
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go makeEnumWorker(ctx, usersChan, &wg)
	}

	var lines []string
	var err error

	if mutateFlag {
		level := mutate.LevelStandard
		switch strings.ToLower(mutateLevelFlag) {
		case "extended":
			level = mutate.LevelExtended
		case "full":
			level = mutate.LevelFull
		}
		lines, err = mutate.GenerateFromFile(namesFileFlag, level, logger)
		if err != nil {
			return fmt.Errorf("mutation error: %w", err)
		}
		logger.Log.Infof("Mutation complete: Generated %d usernames", len(lines))
		if opsecMode {
			logger.Log.Infof("OPSEC mode enabled: shuffling generated usernames")
			shuffleSlice(lines)
		}
	} else if opsecMode && usernamelist != "-" {
		lines, err = loadAndShuffleFile(usernamelist)
		if err != nil {
			logger.Log.Error(err.Error())
			return err
		}
	} else {
		var scanner *bufio.Scanner
		if usernamelist != "-" {
			file, err := os.Open(usernamelist)
			if err != nil {
				logger.Log.Error(err.Error())
				return err
			}
			defer file.Close()
			scanner = bufio.NewScanner(file)
		} else {
			scanner = bufio.NewScanner(os.Stdin)
		}
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				lines = append(lines, line)
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Log.Error(err.Error())
		}
	}

	totalLines := len(lines)
	stopProgress := startProgressTicker(totalLines)
	defer stopProgress()

	start := time.Now()

	for _, usernameline := range lines {
		select {
		case <-ctx.Done():
			break
		default:
			username, err := util.FormatUsername(usernameline)
			if err != nil {
				logger.Log.Debugf("[!] %q - %v", usernameline, err.Error())
				continue
			}
			userFull := fmt.Sprintf("%v@%v", username, domain)
			if shouldSkip(userFull) {
				atomic.AddInt32(&counter, 1)
				continue
			}
			applyDelay()
			usersChan <- username
		}
	}
	close(usersChan)
	wg.Wait()

	finalCount := atomic.LoadInt32(&counter)
	finalSuccess := atomic.LoadInt32(&successes)
	logger.Log.Infof("Done! Tested %d usernames (%d valid) in %.3f seconds", finalCount, finalSuccess, time.Since(start).Seconds())
	return nil
}
