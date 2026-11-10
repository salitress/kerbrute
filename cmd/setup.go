package cmd

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/salitress/kerbrute/session"
	"github.com/salitress/kerbrute/util"
	"github.com/spf13/cobra"
)

var (
	domain           string
	domainController string
	logFileName      string
	verbose          bool
	safe             bool
	delay            int
	jitter           int
	threads          int
	stopOnSuccess    bool
	userAsPass       = false

	downgrade        bool
	hashFileName     string

	ntHash           bool
	socksAddr        string
	eType            string
	linux            bool

	lockoutThreshold int
	jsonOutput       bool
	attemptCounts    sync.Map // username (lowercase) -> *int32, attempts made this run

	logger           util.Logger
	kSession         session.KerbruteSession

	// Used for multithreading
	ctx, cancel = context.WithCancel(context.Background())
	counter     int32
	successes   int32
)

func setupSession(cmd *cobra.Command, args []string) {
	// --delay is meant to space out requests one at a time; a thread count > 1
	// would still fire several requests concurrently regardless of the delay,
	// defeating the point, so force single-threaded operation when it's set.
	if delay != 0 {
		threads = 1
	}

	logger = util.NewLogger(verbose, logFileName, jsonOutput)
	kOptions := session.KerbruteSessionOptions{
		Domain:           domain,
		DomainController: domainController,
		Verbose:          verbose,
		SafeMode:         safe,
		HashFilename:     hashFileName,
		Downgrade:        downgrade,
		NTHash:           ntHash,
		SocksAddr:        socksAddr,
		EncryptionType:   eType,
		Linux:            linux,
		Logger:           &logger,
		JSONOutput:       jsonOutput,
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
	if lockoutThreshold > 0 {
		logger.Log.Infof("Lockout threshold set to %d. Will skip a username after %d attempts in this run\n", lockoutThreshold, lockoutThreshold-1)
	}
}

// attemptsExceeded tracks per-username attempts made during this run and
// reports whether the configured --lockout-threshold safety margin has been
// used up. Disabled (always false) when --lockout-threshold is 0.
func attemptsExceeded(username string) bool {
	if lockoutThreshold <= 0 {
		return false
	}
	key := strings.ToLower(username)
	v, _ := attemptCounts.LoadOrStore(key, new(int32))
	n := atomic.AddInt32(v.(*int32), 1)
	// Leave a 1-attempt safety margin below the real lockout threshold.
	return int(n) > lockoutThreshold-1
}

// sleepDelay waits --delay milliseconds, randomised by +/- --jitter percent
// so requests don't land at a perfectly periodic interval.
func sleepDelay() {
	if delay <= 0 {
		return
	}
	j := jitter
	if j < 0 {
		j = 0
	} else if j > 100 {
		j = 100
	}
	d := delay
	if j > 0 {
		spread := delay * j / 100
		if spread > 0 {
			d = delay - spread + rand.Intn(2*spread+1)
			if d < 0 {
				d = 0
			}
		}
	}
	time.Sleep(time.Duration(d) * time.Millisecond)
}

// lockoutLimitReached reports whether --lockout-threshold has already been
// used up for username, WITHOUT consuming an attempt. Lets a scan loop with
// a single fixed target (bruteuser) stop early instead of grinding through
// the rest of a wordlist, sleeping --delay each time, once nothing more can
// be learned.
func lockoutLimitReached(username string) bool {
	if lockoutThreshold <= 0 {
		return false
	}
	v, ok := attemptCounts.Load(strings.ToLower(username))
	if !ok {
		return false
	}
	return int(atomic.LoadInt32(v.(*int32))) > lockoutThreshold-1
}
