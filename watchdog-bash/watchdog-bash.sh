#!/bin/bash

CURL_TMP_RESULT=/tmp/curl_result.txt

WATCHED_IP=127.0.0.1
WATCHED_PORT=8080
WATCHED_NET=$WATCHED_IP:$WATCHED_PORT
CATALINA_BASE=/usr/local/tomcat

trap "rm -f ${CURL_TMP_RESULT}" EXIT

getPidByPort() {
    pid_raw=`ss -lnpt|grep ":$WATCHED_PORT "|awk '{print $6}'|awk -F "pid=" '{print $2}'|awk -F "," '{print $1}'`

    [[ -n $pid_raw ]] && {
        space_regex=".* .*"
        if [[ $pid_raw =~ $space_regex ]] ; then
            pid=`echo $pid_raw|awk '{print $1}'`
        else
            pid=$pid_raw
        fi
    }
}

healthCheck() {
    # limit whole check time in 8 seconds and connect time in 2 seconds
    curl -s --connect-timeout 2 --max-time 8 -o $CURL_TMP_RESULT $WATCHED_NET && HEALTH_STATUS="success" || HEALTH_STATUS="fail"
}

watchdogTomcat() {
    # INITIAL OF WATCHDOG
    # go to watchdog logic when conditions down satisfied
    #   - pid exist
    #   - first health check status success
    while : ; do
        [[ -n $pid ]] && {
            healthCheck
            [[ $HEALTH_STATUS -eq "success" ]] && {
                echo "healthCheck >>> notify systemd READY=1" # debug
                systemd-notify --pid=$pid --ready
                break
            }
        } || {
            getPidByPort
            #echo "getPidByPort >>> PID_RAW=$pid_raw PID:$pid"             # debug
        }
    done

    # WATCHDOG START
    while : ; do
        interval=$(($WATCHDOG_USEC / $((2 * 1000000))))

        healthCheck

        if [[ $HEALTH_STATUS -eq "success" ]] ; then
            #echo "watchdog detect success" # debug
            systemd-notify --pid=$pid WATCHDOG=1
            sleep ${interval}
        else
            #echo "watchdog detect failed" # debug
            sleep 1
        fi
    done
}


${CATALINA_BASE}/bin/catalina.sh start
watchdogTomcat &
