# Right Size Guide (rsg)

Right Size Guide (`rsg`) is a simple CLI tool that provides you with memory and 
CPU recommendations for your application. While, in containerized setups and 
there especially in Kubernetes-land, components such as the
[VPA](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) 
exist that could be used to extract similar recommendations, they are: 

1. limited to a certain orchestrator (Kubernetes), and 
1. rather complicated to use, since they come with a number of dependencies.
   
With `rsg` we want to do the opposite: an easy to use tool with zero dependencies
that can be used with and for any container orchestrator, including Kubernetes,
Amazon ECS, HashiCorp Nomad, and even good old Apache Mesos+Marathon.

## Install it

Download the [latest binary](https://github.com/mhausenblas/rsg/releases/latest) 
for Linux (Intel or Arm) and macOS.

For example, to install `rsg` from binary on macOS you could do the following:

```sh
curl -L https://github.com/mhausenblas/rsg/releases/latest/download/rsg_darwin_amd64.tar.gz \
    -o rsg.tar.gz && \
    tar xvzf rsg.tar.gz rsg && \
    mv rsg /usr/local/bin && \
    rm rsg*
```

## Use it

Imagine you have an application called `test`. It's a binary executable that 
exposes an HTTP API on `:8080/ping`. Now, this is how you can use `rsg` to 
figure out how much memory and/or CPU your app requires:

```sh
$ rsg --target test/test --api-path /ping --api-port 8080
2020/02/15 12:40:42 Launching test/test for idle state resource usage assessment
2020/02/15 12:40:42 Trying to determine idle state resource usage (no external traffic)
2020/02/15 12:40:44 Idle state assessment of test/test completed
2020/02/15 12:40:44 Found idle state resource usage. MEMORY: 7684kB CPU: 7ms (user)/7ms (sys)
2020/02/15 12:40:44 Launching test/test for peak state resource usage assessment
2020/02/15 12:40:44 Trying to determine peak state resource usage using 127.0.0.1:8080/ping
2020/02/15 12:40:45 Starting to hammer the endpoint http://127.0.0.1:8080/ping every 10ms
2020/02/15 12:40:49 Peak state assessment of test/test completed
2020/02/15 12:40:49 Found peak state resource usage. MEMORY: 13824kB CPU: 68ms (user)/62ms (sys)
{
 "idle": {
  "memory_in_bytes": 7684096,
  "cpuuser_in_usec": 7926,
  "cpusys_in_usec": 7536
 },
 "peak": {
  "memory_in_bytes": 13824000,
  "cpuuser_in_usec": 68745,
  "cpusys_in_usec": 62138
 }
}
```

You can also specify a file explicitly and suppress the status updates, like so:

```sh
$ go run main.go --target test/test \
                 --api-path /ping --api-port 8080 \
                 --export-findings ./results.json 2>/dev/null
```

Then, you could pull out specific results, say, using `jq`:

```sh
$ cat results.json | jq .peak.cpuuser_in_usec
76174
```


## Background

Based on an [informal query on Twitter](https://twitter.com/mhausenblas/status/1225855388584730624) these tools already provide similar functionality:

- [time](http://man7.org/linux/man-pages/man1/time.1.html)
- [perf](http://www.brendangregg.com/perf.html)
- [DTrace](http://www.brendangregg.com/DTrace/cputimes)
