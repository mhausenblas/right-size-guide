The right size guide (`rsg`) is a simple CLI tool that provides you with memory and 
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

Download the [latest binary](https://github.com/mhausenblas/right-size-guide/releases/latest) 
for Linux (Intel or Arm) and macOS.

For example, to install `rsg` from binary on macOS you could do the following:

```sh
curl -L https://github.com/mhausenblas/right-size-guide/releases/latest/download/rsg_darwin_amd64.tar.gz \
    -o rsg.tar.gz && \
    tar xvzf rsg.tar.gz rsg && \
    mv rsg /usr/local/bin && \
    rm rsg*
```

## Prior art

Based on an [informal query on Twitter](https://twitter.com/mhausenblas/status/1225855388584730624) these tools already provide similar functionality:

- [time](http://man7.org/linux/man-pages/man1/time.1.html)
- [perf](http://www.brendangregg.com/perf.html)
- [DTrace](http://www.brendangregg.com/DTrace/cputimes)
- Facebook [senpai](https://github.com/facebookincubator/senpai)
- Kubernetes [VPA](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) 
