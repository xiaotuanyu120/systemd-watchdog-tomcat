### systemd soft watchdog for tomcat
``` bash
git clone https://github.com/xiaotuanyu120/systemd-watchdog-tomcat.git
cd systemd-watchdog-tomcat

mv watchdog-tomcat.sh /usr/local/bin/watchdog-tomcat
chmod 755 /usr/local/bin/watchdog-tomcat
mv watchdog-tomcat.service /usr/lib/systemd/system/tomcat.service

# make sure tomcat running permission
id tomcat >/dev/null 2>&1 || useradd -r -s /sbin/nologin tomcat 
chown -R tomcat.tomcat /usr/local/tomcat
```
