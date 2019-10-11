#!/bin/bash

CURL_TMP_RESULT=/tmp/curl_result.txt

WATCHED_IP=127.0.0.1
WATCHED_PORT=8080
WATCHED_NET=$WATCHED_IP:$WATCHED_PORT
CATALINA_BASE=/usr/local/tomcat

trap "rm -f ${CURL_TMP_RESULT}" EXIT

getPidByPort() {
    port=$1
    pid_raw=`ss -lnpt|grep ":$WATCHED_PORT "|awk '{print $6}'|awk -F "pid=" '{print $2}'|awk -F "," '{print $1}'`

    if [[ -n $pid_raw ]] ; then
        space_regex=".* .*"
        if [[ $pid_raw =~ $space_regex ]] ; then
            pid=`echo $pid_raw|awk '{print $1}'`
        else
            pid=$pid_raw
        fi
    else
       pid="null"
    fi
}

watchdogTomcat() {
    while : ; do
        getPidByPort $WATCHED_PORT
        [[ $pid != "null" ]] && systemd-notify --ready ; sleep 2 ; break
    done

    while : ; do
        FAIL=0
        interval=$(($WATCHDOG_USEC / $((2 * 1000000))))
        curl -s -o $CURL_TMP_RESULT $WATCHED_NET || FAIL=1

        if [[ $FAIL -eq 0 ]] ; then
            echo "watchdog detect success" # debug
            systemd-notify WATCHDOG=1
            sleep ${interval}
        else
            echo "watchdog detect failed" # debug
            sleep 1
        fi
    done
}



watchdogTomcat &
eval exec ${CATALINA_BASE}/bin/catalina.sh start
