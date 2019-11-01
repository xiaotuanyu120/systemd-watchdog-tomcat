package main

import (
    "bytes"
    "errors"
    "flag"
    "fmt"
    "io"
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
    /*Execute Application
      1. run app and set group pid for the forked child process
      2. wait app start up*/
    cmd := exec.Command(application)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    err = cmd.Run()
    if err != nil {
        fmt.Println(err)
        pid = 0
        return pid, err
    }
    pgid := cmd.Process.Pid
    cmd.Wait()

    /*Get pid of JVM
      get pid of child process by filter ps result using group pid*/
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
        return pid, err
    }
    err = grep.Start()
    if err != nil {
        fmt.Println(err)
        pid = 0
        return pid, err
    }
    go func() {
        defer pw.Close()
        ps.Wait()
    }()
    defer pr.Close()
    grep.Wait()

    res := strings.TrimSpace(out.String())
    pid, _ = strconv.Atoi(strings.Split(res, " ")[0])
    return pid, nil
}

func healthCheck(check_url string, timeout time.Duration) (bool, error) {
    var netClient = &http.Client{
        Timeout: time.Second * timeout,
    }
    resp, err := netClient.Get(check_url)
    if err != nil {
        return false, err
    }
    if resp.StatusCode != 200 {
        return false, errors.New("health check status is not 200")
    }
    defer resp.Body.Close()
    return true, nil
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
    healthcheck_timeout := flag.Duration("healthcheck-timeout", 5, "Timeout for healthcheck when service is running")
    initialcheck_timeout := flag.Duration("initialcheck-timeout", 5, "Timeout for initialcheck when service is boot up")
    fail_max := flag.Int("fail-max", 20, "max continued failed time")
    flag.Parse()

    if !fileExists(*app) {
        fmt.Printf("app [%s] should exist and is an executable file", app)
        os.Exit(1)
    }

    check_url := fmt.Sprintf("%s://%s:%s/%s", *scheme, *ip, *port, *path)

    // RUN APPLICATION
    pid, err := runWatchedApp(*app)
    if err != nil {
        fmt.Printf("application run error: %i\n", err)
        os.Exit(2)
    } else {
        daemon.SdNotify(false, fmt.Sprintf("MAINPID=%d", pid))
    }

    /* WATCHDOG INITIAL
       start watchdog when
       1. first health check success*/
    for {
        _, err := healthCheck(check_url, *initialcheck_timeout)
        if err == nil {
            daemon.SdNotify(false, daemon.SdNotifyReady)
            fmt.Println("WATCHDOG INITIALIZING: program is ok, watchdog is ready")
            break
        } else {
            fmt.Printf("WATCHDOG INITIALIZING: program is not ok, watchdog is waiting\nINITIAL ERROR: %s\n", err.Error())
        }
        time.Sleep(1000 * time.Millisecond)
    }

    // WATCHDOG START
    var wg sync.WaitGroup
    wg.Add(1)
    go func(check_url string) {
        watchdog_usec, _ := strconv.ParseFloat(os.Getenv("WATCHDOG_USEC"), 64)

        var wd_fail bool
        var wd_usec, check_time_spent float64
        var wd_interval, continue_fail int

        wd_usec = watchdog_usec / (2 * 1000000)

        for {
            // send watchdog signal
            if wd_fail == false {
                daemon.SdNotify(false, daemon.SdNotifyWatchdog)
                fmt.Printf("WATCHDOG STATUS: activate; LAST_CHECK_TIME_SPENT: %f; LAST_SLEEP_TIME: %d\n", check_time_spent, wd_interval)
            }

            // check and change the watchdog failed state
            check_start := time.Now()
            check_success, err := healthCheck(check_url, *healthcheck_timeout)
            if check_success == true {
                continue_fail = 0
            } else {
                continue_fail += 1
            }
            if continue_fail > *fail_max {
                wd_fail = true
            } else {
                wd_fail = false
            }
            check_time_spent = time.Since(check_start).Seconds()

            if wd_fail == false {
                if check_success == false {
                    // add your alert logic here
                    fmt.Printf("CHECK STATUS: failed; ERROR: %s\n", err.Error())
                }
                wd_interval = int(wd_usec - check_time_spent + 0.5)
                time.Sleep(time.Duration(wd_interval) * 1000 * time.Millisecond)
            } else {
                // sleep until systemd watchdog exceed limit
                fmt.Printf("Watchdog change to failed state, because continued failed time is exceed fail-max limit: %d\n", *fail_max)
                time.Sleep(time.Duration(int(wd_usec*2-check_time_spent+0.5)) * 1000 * time.Millisecond)
            }
        }
    }(check_url)
    wg.Wait()
}