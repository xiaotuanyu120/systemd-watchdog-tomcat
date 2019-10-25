package main

import (
    "bytes"
    "errors"
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

func healthCheck(check_url string) (bool, error) {
    resp, err := http.Get(check_url)
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
    flag.Parse()

    if !fileExists(*app) {
        log.Fatal("app should be not empty")
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
        _, err := healthCheck(check_url)
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
        var wd_interval int

        wd_usec = watchdog_usec / (2 * 1000000)

        for {
            if wd_fail == false {
                daemon.SdNotify(false, daemon.SdNotifyWatchdog)
                fmt.Printf("CHECK STATUS: success; TIME_SPENT: %f; SLEEP: %d\n", check_time_spent, wd_interval)
            }

            check_start := time.Now()
            check_result, err := healthCheck(check_url)
            wd_fail = !check_result
            check_time_spent = time.Since(check_start).Seconds()

            if wd_fail == false {
                wd_interval = int(wd_usec - check_time_spent + 0.5)
                time.Sleep(time.Duration(wd_interval) * 1000 * time.Millisecond)
            } else {
                fmt.Printf("CHECK STATUS: failed; ERROR: %s\n", err.Error())
                time.Sleep(1000 * time.Millisecond)
            }
        }
    }(check_url)
    wg.Wait()
}