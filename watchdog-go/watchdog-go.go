package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/daemon"
)

func runWatchedApp(application string) (pid int, err error) {
	/*Execute Application*/
	cmd := exec.Command(application)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		pid = 0
		return
	}
	pgid := cmd.Process.Pid
	cmd.Wait()

	/*Get pid of JVM*/
	grep := exec.Command("grep", strconv.Itoa(pgid))
	ps := exec.Command("ps", "axo", "pid,pgid,comm")

	var out bytes.Buffer
	pr, pw := io.Pipe()

	ps.Stdout = pw
	grep.Stdin = pr
	grep.Stdout = &out

	err = ps.Start()
	if err != nil {
		fmt.Println(err)
		pid = 0
		return
	}
	err = grep.Start()
	if err != nil {
		fmt.Println(err)
		pid = 0
		return
	}
	go func() {
		defer pw.Close()
		ps.Wait()
	}()
	grep.Wait()

	res := strings.TrimSpace(out.String())
	pid, _ = strconv.Atoi(strings.Split(res, " ")[0])
	return
}

func healthCheck(check_url string) (err error) {
	resp, err := http.Get(check_url)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func main() {
	// GET ALL FLAGS
	app := flag.String("app", "", "path to the app should be run")
	scheme := flag.String("scheme", "http", "scheme for health check,\n    EXAMPLE: 'scheme://ip:port/path'")
	ip := flag.String("ip", "127.0.0.1", "ip address for health check,\n    EXAMPLE: 'scheme://ip:port/path'")
	port := flag.String("port", "80", "port for health check,\n    EXAMPLE: 'scheme://ip:port/path'")
	path := flag.String("path", "", "path for health check,\n    EXAMPLE: 'scheme://ip:port/path'")
	flag.Parse()

	if !fileExists(*app) {
		log.Fatal("app should be not empty")
		os.Exit(1)
	}

	check_url := fmt.Sprintf("%s://%s:%s/%s", *scheme, *ip, *port, *path)

	// RUN APPLICATION
	pid, err := runWatchedApp(*app)
	if err != nil {
		fmt.Println("application run error: %i", err)
		os.Exit(2)
	}

	// WATCHDOG INITIAL
	for {
		err := healthCheck(check_url)
		if err == nil {
			state := fmt.Sprintf("MAINPID=%d\n%s", pid, daemon.SdNotifyReady)
			daemon.SdNotify(false, state)
			fmt.Println("watchdog and program is ready")

			break
		}
		time.Sleep(1000 * time.Millisecond)
	}

	// WATCHDOG START
	var wg sync.WaitGroup
	wg.Add(1)
	go func(check_url string) {
		watchdog_usec, _ := strconv.Atoi(os.Getenv("WATCHDOG_USEC"))
		wd_interval := watchdog_usec / (2 * 1000000)

		for {
			wd_fail := true

			err := healthCheck(check_url)
			if err != nil {
				wd_fail = false
			}

			if wd_fail == true {
				fmt.Println("watchdog check success")
				daemon.SdNotify(false, daemon.SdNotifyWatchdog)
				time.Sleep(time.Duration(wd_interval) * 1000 * time.Millisecond)
			} else {
				fmt.Println("watchdog check failed")
				time.Sleep(1000 * time.Millisecond)
			}
		}
	}(check_url)
	wg.Wait()
}
