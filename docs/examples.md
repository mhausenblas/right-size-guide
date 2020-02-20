In this section we show how to apply `rsg` findings in the context of container 
orchestrators or other related environments, such as serverless compute engines.

### Kubernetes

In Kubernetes, the [Quality of Service (QoS) ](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/) 
of a pod is defined via the resource requests and limits. The former being an 
input for the scheduler to find a fitting node to launch the pod and the latter
can be a cause to terminate any container in a pod. For example,
if a container [consumes memory beyond its limit](https://kubernetes.io/docs/tasks/configure-pod-container/assign-memory-resource/)
it is [OOM](https://www.kernel.org/doc/gorman/html/understand/understand016.html) killed.

But how to arrive at "good" values for the resource requests and limits, which should
be the same BTW if you want to have deterministic behavior? `rsg` to the rescue …

Let's say you run `rsg` on your app and get the following results (made up, but hey):

```json
{
 "idle": {
  "memory_in_bytes": 10123456,
  "cpuuser_in_usec": 2000,
  "cpusys_in_usec": 13000
 },
 "peak": {
  "memory_in_bytes": 25042987,
  "cpuuser_in_usec": 4000,
  "cpusys_in_usec": 87666
 }
}
```

You would then plug it into the Kubernetes manifest like so:

```yaml
...
spec:
  containers:
  - name: someapp
    image: theimage:sometag
    resources:
      limits:
        memory: "30M"
        cpu: "900m"
      requests:
        memory: "30M"
        cpu: "900m"
```

!!! tip
    Above we used the peak value of the `rsg` finding `memory_in_bytes` which is
    `25,042,987` and padded it a little, arriving at `30M`, which is the [Kubernetes
    way of saying](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) 
    "can I have 30 MB please".


### AWS Fargate

In EKS on Fargate, one must observe the [pod resource allocations](https://docs.aws.amazon.com/eks/latest/userguide/fargate-pod-configuration.html).
You can see what happens if you don't do this in [this Gist](https://gist.github.com/mhausenblas/56db56d63dad78fc4e81108da49f28b2).

_TBD_

### Amazon ECS

_TBD_