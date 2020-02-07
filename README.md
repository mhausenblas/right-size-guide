# Right Size Guide (rsg)

Right Size Guide (`rsg`) is a simple CLI tool that provides you with memory and CPU recommendations for your application. While, in containerized setups and there espectially in Kubernetes-land, things like the [VPA](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) exist that could be used to extract similar recommendations, they are: 1. limited to a certain orchestrator (Kubernetes), and 2. rather complicated to use. With `rsg` we want to do the opposite: an easy to use tool that can be used with and for any container orchestrator.

## Install it

TBD.

## Use it

Imagine you have an application called `foo`. It's a binary executable that can be run directly (for example, it doesn't depend on a runtime such as a JVM). Now, this is how you can use `rsg` to figure out how much memory and/or CPU your app requires:

```sh
$ rsg --target /some/path/foo --api-path /test --api-port 8080 --export-findings ./foo-resource-usage.txt
2020-03-03T10:42:10 Launching /some/path/foo for idle state resource usage assessment
2020-03-03T10:42:13 Trying to determine idle state resource usage (no external traffic)
2020-03-03T10:42:59 Found idle state resource usage: MEMORY = 533 MB and CPU = 2900 milli
2020-03-03T10:43:03 Launching load test for /some/path/foo for peak state resource usage assessment
2020-03-03T10:53:22 Found peak state resource usage: MEMORY = 800 MB and CPU = 4000 milli
```
