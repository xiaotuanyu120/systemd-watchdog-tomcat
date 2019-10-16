package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/coreos/go-systemd/daemon"
)

func runWatchedApp(application string) (pid int, err error) {
	cmd := exec.Command(application)
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		pid = 0
		return
	}
	pid = cmd.Process.Pid
	return
}

func main() {
	if len(os.Args) != 2 {
		os.Exit(1)
	}
	app := os.Args[1]

	pid, err := runWatchedApp(app)
	if err != nil {
		fmt.Println("erro", err)
		os.Exit(2)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		unsetEnv := false
		watchdog_usec, _ := strconv.Atoi(os.Getenv("WATCHDOG_USEC"))
		wd_interval := watchdog_usec / (2 * 1000000)

		//state := fmt.Sprintf("MAINPID=%d\n%s", pid, daemon.SdNotifyWatchdog)

		fmt.Println("PID: ", pid)
		daemon.SdNotify(unsetEnv, fmt.Sprintf("MAINPID=%d", pid))
		daemon.SdNotify(unsetEnv, fmt.Sprintf(daemon.SdNotifyReady))
		for {
			wd_fail := 0

			if wd_fail == 0 {
				fmt.Println("watchdog check success")
				daemon.SdNotify(unsetEnv, fmt.Sprintf(daemon.SdNotifyWatchdog))
				time.Sleep(time.Duration(wd_interval) * 1000 * time.Millisecond)
			} else {
				fmt.Println("watchdog check failed")
				time.Sleep(1000 * time.Millisecond)
			}
		}
	}()
	wg.Wait()
}
