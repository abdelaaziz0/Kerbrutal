package cmd

import (
	"sync"
	"sync/atomic"
	"time"
)

var (
	lockoutCount  int32
	successStreak int32
	maxBackoff    = int64(30000) // 30 seconds

	backoffMu    sync.Mutex
	backoffDelay int64 // protected exclusively by backoffMu

	adaptiveBackoff bool // controlled by --adaptive-backoff flag
)

func recordLockout() {
	atomic.AddInt32(&lockoutCount, 1)
	atomic.StoreInt32(&successStreak, 0)

	if !adaptiveBackoff {
		logger.Log.Warningf("[!] Account lockout detected (#%d)! (use --adaptive-backoff to slow down automatically)",
			atomic.LoadInt32(&lockoutCount))
		return
	}

	backoffMu.Lock()
	defer backoffMu.Unlock()
	if backoffDelay == 0 {
		backoffDelay = 1000
	} else {
		backoffDelay *= 2
	}
	if backoffDelay > maxBackoff {
		backoffDelay = maxBackoff
	}
	logger.Log.Warningf("[!] Account lockout detected (#%d)! Adaptive backoff → %dms",
		atomic.LoadInt32(&lockoutCount), backoffDelay)
}

func recordSuccess() {
	if !adaptiveBackoff {
		return
	}
	streak := atomic.AddInt32(&successStreak, 1)
	if streak >= 10 {
		backoffMu.Lock()
		defer backoffMu.Unlock()
		if backoffDelay > 0 {
			backoffDelay /= 2
			if backoffDelay < 500 {
				backoffDelay = 0
			}
			if backoffDelay > 0 {
				logger.Log.Infof("[*] Backoff reduced → %dms after %d consecutive successes", backoffDelay, streak)
			} else {
				logger.Log.Infof("[*] Backoff cleared after %d consecutive successes", streak)
			}
		}
		atomic.StoreInt32(&successStreak, 0)
	}
}

func getAdaptiveDelay() time.Duration {
	backoffMu.Lock()
	defer backoffMu.Unlock()
	return time.Duration(backoffDelay) * time.Millisecond
}
