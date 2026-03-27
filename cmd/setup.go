package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/abdelaaziz0/kerbrutal/session"
	"github.com/abdelaaziz0/kerbrutal/transport"
	"github.com/abdelaaziz0/kerbrutal/util"
	"github.com/spf13/cobra"
)

var (
	domain           string
	domainController string
	logFileName      string
	verbose          bool
	safe             bool
	delay            int
	threads          int
	stopOnSuccess    bool
	userAsPass       = false

	downgrade    bool
	hashFileName string

	validUsersFileName string
	validUsersWriter   *os.File
	validUsersMu       sync.Mutex

	proxyFlag string
	proxyUser string
	proxyPass string

	logger   util.Logger
	kSession session.KerbruteSession


	ctx, cancel = context.WithCancel(context.Background())
	counter     int32
	successes   int32
)

func setupSession(cmd *cobra.Command, args []string) {
	logger = util.NewLogger(verbose, logFileName, jsonOutput)

	if delay != 0 && threads != 1 {
		threads = 1
		logger.Log.Infof("Delay set — forcing single thread mode")
	}

	if validUsersFileName != "" {
		f, err := os.OpenFile(validUsersFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			logger.Log.Errorf("Could not open valid-users file: %v", err)
			os.Exit(1)
		}
		validUsersWriter = f
		logger.Log.Infof("Saving valid results to %s", validUsersFileName)
	}

	if resumeFile != "" {
		loadResumeState(resumeFile)
		initResumeWriter(resumeFile)
		logger.Log.Infof("Resume mode enabled: %s", resumeFile)
	}

	if opsecMode {
		logger.Log.Infof("OPSEC mode enabled: shuffled wordlists + jittered delays")
	}

	if jsonOutput {
		logger.Log.Infof("JSON output mode enabled (results on stdout)")
	}

	var d transport.KDCDialer
	if proxyFlag != "" {
		socksDialer, err := transport.ParseProxyURL(proxyFlag, 5 * time.Second)
		if err != nil {
			logger.Log.Errorf("Invalid --proxy value: %v", err)
			os.Exit(1)
		}
		if proxyUser != "" {
			socksDialer.Username = proxyUser
			socksDialer.Password = proxyPass
		}
		d = socksDialer
		logger.Log.Infof("SOCKS5 proxy configured: %s", proxyFlag)
	}

	kOptions := session.KerbruteSessionOptions{
		Domain:           domain,
		DomainController: domainController,
		Verbose:          verbose,
		SafeMode:         safe,
		HashFilename:     hashFileName,
		Downgrade:        downgrade,
		Dialer:           d,
	}
	k, err := session.NewKerbruteSession(kOptions)
	if err != nil {
		logger.Log.Error(err)
		os.Exit(1)
	}
	kSession = k

	logger.Log.Info("Using KDC(s):")
	for _, v := range kSession.Kdcs {
		logger.Log.Infof("\t%s\n", v)
	}
	if delay != 0 {
		logger.Log.Infof("Delay set. Using single thread and delaying %dms between attempts\n", delay)
	}
}

func writeValidUser(userFull string) {
	if validUsersWriter == nil {
		return
	}
	validUsersMu.Lock()
	defer validUsersMu.Unlock()
	validUsersWriter.WriteString(fmt.Sprintf("%s\n", userFull))
}

func closeValidUsersFile() {
	if validUsersWriter != nil {
		validUsersWriter.Close()
	}
}
