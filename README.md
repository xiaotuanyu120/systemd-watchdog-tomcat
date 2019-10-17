### systemd soft watchdog for tomcat
#### bash version
this is for shell script, but must be run under root
``` bash
git clone https://github.com/xiaotuanyu120/systemd-watchdog-tomcat.git
cd systemd-watchdog-tomcat

cp watchdog-bash/watchdog-bash.sh /usr/local/bin/watchdog-bash
chmod 755 /usr/local/bin/watchdog-bash
cp watchdog-bash.service /usr/lib/systemd/system/tomcat.service
```
#### go version
this is for go, and can run under nonroot
``` bash
git clone https://github.com/xiaotuanyu120/systemd-watchdog-tomcat.git
cd systemd-watchdog-tomcat/watchdog-go

go build ./watchdog-go

cp watchdog-go /usr/local/bin/watchdog-go
chmod 755 /usr/local/bin/watchdog-go
cp watchdog-go.service /usr/lib/systemd/system/app.service
```