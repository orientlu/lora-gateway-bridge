#!/bin/bash
# by orientlu

#set -x
# target ip
#host="129.204.140.64:9877"
host="127.0.0.1:9878"
# sample time
seconds=60
# open a http for report
rep_port=9879
# app bin path
bin="./build/lora-gateway-bridge"
# save profile prefix
prefix="gw"
# proxy
proxy=""


#######################################################

prefix="${prefix}-$(date +%y%m%d-%H:%M:%S)"

if [[ -z "$1" ]];then
    echo -e "usage:\n\t$0 cmd\n\\ntcmd:cpu|heap|block|trace\n"
fi

if [[ -n "$1" && "$1" == "cpu" ]]; then
    go tool pprof -http ":${rep_port}"  http://${host}/debug/pprof/profile?seconds=${seconds}
fi

if [[ -n "$1" && "$1" == "cpus" ]]; then
    curl -o ${prefix}_cpu_profile.out http://${host}/debug/pprof/profile?seconds=${seconds} -x "${proxy}"
    go tool pprof -http=":${rep_port}" ${bin} ./${prefix}_cpu_profile.out
fi


if [[ -n "$1" && "$1" == "heap" ]]; then
    curl -o ${prefix}_heap_profile.out http://${host}/debug/pprof/heap?seconds=${seconds} -x "${proxy}"
    go tool pprof -http=":${rep_port}" ${bin} ./${prefix}_heap_profile.out
    #go tool pprof -http ":${rep_port}"  http://${host}/debug/pprof/heap?seconds=${seconds}
fi

if [[ -n "$1" && "$1" == "block" ]]; then
    curl -o ${prefix}_block_profile.out http://${host}/debug/pprof/block?seconds=${seconds} -x "${proxy}"
    go tool pprof -http=":${rep_port}" ${bin} ./${prefix}_block_profile.out
fi

if [[ -n "$1" && "$1" == "trace" ]]; then
    curl -o ${prefix}_trace.out http://${host}/debug/pprof/trace?seconds=${seconds} -x "${proxy}"
    go tool trace -http=":${rep_port}" ${bin} ./${prefix}_trace.out
fi
