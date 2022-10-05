## systemd soft watchdog for tomcat
Normally systemd will monitor service in process level. **systemd-watchdog-tomcat provide more than that, it will monitor service using your customized healthcheck logic.** This is based on systemd's watchdog.

> want to know more about systemd's watchdog, check out the links below.
> - [systemd.service #WatchdogSec](https://www.freedesktop.org/software/systemd/man/systemd.service.html#WatchdogSec=)
> - [sd_notify #Description](https://www.freedesktop.org/software/systemd/man/sd_notify.html#Description)

> 两篇中文博客文档
> - [systemd-watchdog之旅1 - 脚本篇](https://blog.xiaotuanyu.com/linux/advance/systemd_2.1.0_watchdog_for_tomcat.html)
> - [systemd-watchdog之旅2 - golang篇](https://blog.xiaotuanyu.com/linux/advance/systemd_2.1.1_watchdog_for_tomcat_error_of_nonroot_user.html)

### How to use systemd-watchdog-tomcat
First, clone systemd-watchdog-tomcat and build it
``` bash
git clone https://github.com/xiaotuanyu120/systemd-watchdog-tomcat.git
cd systemd-watchdog-tomcat

go build ./watchdog-tomcat.go

cp watchdog-tomcat /usr/local/bin/watchdog-tomcat
chmod 755 /usr/local/bin/watchdog-tomcat
```

Then, prepare your systemd unit file, here is a example
```
[Unit]
Description=watchdog - golang
 
[Service]
Type=notify
User=tomcat
Group=tomcat
Environment=JAVA_HOME=/usr/local/jdk
ExecStart=/usr/local/bin/watchdog-tomcat \
    -app=/usr/local/tomcat/bin/startup.sh \
    -scheme=http \
    -ip=127.0.0.1 \
    -path=/ \
    -port=8080
ExecStop=/bin/kill -9 $MAINPID
WatchdogSec=30
NotifyAccess=all
Restart=always
RestartSec=5s
TimeoutStartSec=2min
 
[Install]
WantedBy=multi-user.target
```

Important point
1. `Type` must be `notify`; systemd will wait singnal `READY=1` to make service transfer to start up status. [click here for more details](https://www.freedesktop.org/software/systemd/man/systemd.service.html#Type=)
2. `User` and `Group` support normal user here, root is also okay. if you use root, you can also use bash version below
3. `ExecStart`, watchdog-tomcat provides options below:
  - `-app`, normally its tomcat startup shell scripts
  - `-scheme`, scheme of health check url, default is "http", only can use "http" or "https"
  - `-ip`, ip or domain of health check url, default is "127.0.0.1", also can use "myhealthcheck.com"
  - `-path`, health check url's path, for example "scheme://ip:port/path", default is "/"
  - `-port`, health check url's port, default is "80"
4. `WatchdogSec`, Configures the watchdog timeout for a service. The time configured here will be passed to the executed service process in the WATCHDOG_USEC= environment variable. [click here for more details](https://www.freedesktop.org/software/systemd/man/systemd.service.html#WatchdogSec=)
5. `NotifyAccess=all`, cause we will configure java process to MAINPID, so here we config it to all, that means watchdog can sent signal to control main process's status.

Finally, try to reload service and start it
``` bash
systemctl daemon-reload
systemctl restart watchdog-tomcat.service
```
also, you can check the logs using `journalctl -xefu watchdog-tomcat`
``` 
-- Unit watchdog-tomcat.service has begun starting up.
Oct 25 15:17:22 localhost.localdomain watchdog-go[10453]: WATCHDOG INITIALIZING: program is not ok, watchdog is waiting
Oct 25 15:17:22 localhost.localdomain watchdog-go[10453]: INITIAL ERROR: Get http://127.0.0.1:8080/: dial tcp 127.0.0.1:8080: connect: connection refused
Oct 25 15:17:23 localhost.localdomain watchdog-go[10453]: WATCHDOG INITIALIZING: program is not ok, watchdog is waiting
Oct 25 15:17:23 localhost.localdomain watchdog-go[10453]: INITIAL ERROR: Get http://127.0.0.1:8080/: dial tcp 127.0.0.1:8080: connect: connection refused
Oct 25 15:17:25 localhost.localdomain watchdog-go[10453]: WATCHDOG INITIALIZING: program is ok, watchdog is ready
Oct 25 15:17:25 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.000000; SLEEP: 0
Oct 25 15:17:25 localhost.localdomain systemd[1]: Started watchdog test.
-- Subject: Unit watchdog-tomcat.service has finished start-up
-- Defined-By: systemd
-- Support: http://lists.freedesktop.org/mailman/listinfo/systemd-devel
-- 
-- Unit watchdog-tomcat.service has finished starting up.
-- 
-- The start-up result is done.
Oct 25 15:17:40 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.008790; SLEEP: 15
Oct 25 15:17:55 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.005073; SLEEP: 15
Oct 25 15:18:10 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.005967; SLEEP: 15
Oct 25 15:18:25 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.003074; SLEEP: 15
Oct 25 15:18:41 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.003004; SLEEP: 15
Oct 25 15:18:56 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.003917; SLEEP: 15
Oct 25 15:19:11 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.002733; SLEEP: 15
Oct 25 15:19:26 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.005026; SLEEP: 15
Oct 25 15:19:41 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.004405; SLEEP: 15
Oct 25 15:19:56 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.003201; SLEEP: 15
Oct 25 15:20:11 localhost.localdomain watchdog-go[10453]: CHECK STATUS: success; TIME_SPENT: 0.002929; SLEEP: 15
```

> If you run service under root user, you can also use the bash version.  
> ``` bash
> git clone https://github.com/xiaotuanyu120/systemd-watchdog-tomcat.git
> cd systemd-watchdog-tomcat
> 
> cp script/watchdog-bash.sh /usr/local/bin/watchdog-bash
> chmod 755 /usr/local/bin/watchdog-bash
> cp script/watchdog-bash.service /usr/lib/systemd/system/tomcat.service
> ```
