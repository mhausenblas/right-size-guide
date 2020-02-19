You can use `rsg` for both idle and peak resource usage assessment. The findings 
are presented in a simple JSON format, friendly for usage in a pipeline or 
in [OpenMetrics](https://openmetrics.io/) for ingestion in Prometheus. Status
updates are written to `stderr` so you can suppress them if you like.

```sh
$ rsg -h
Usage:
 rsg --target $BINARY
 [--api-path $HTTP_URL_PATH --api-port $HTTP_PORT --peak-delay $TIME_MS --sampletime-idle $TIME_SEC --sampletime-peak $TIME_SEC --export-findings $FILE --output json|openmetrics]
Example usage:
 rsg --target test/test --api-path /ping --api-port 8080 2>/dev/null
Arguments:
  -api-path string
        [OPTIONAL] The URL path component of the HTTP API to use for peak resource usage assessment
  -api-port string
        [OPTIONAL] The TCP port of the HTTP API to use for peak resource usage assessment
  -delay-peak int
        [OPTIONAL] The time in milliseconds to wait between two consecutive HTTP GET requests for peak resource usage assessment (default 10)
  -export-findings string
        [OPTIONAL] The filesystem path to export findings to; if not provided the results will be written to stdout
  -output string
        [OPTIONAL] The output format, valid values are 'json' and 'openmetrics' (default "json")
  -sampletime-idle int
        [OPTIONAL] The time in seconds to perform idle resource usage assessment (default 2)
  -sampletime-peak int
        [OPTIONAL] The time in seconds to perform peak resource usage assessment (default 10)
  -target string
        The filesystem path of the binary or script to assess
  -version
        Print the version of rsg and exit
```

### Assessing the idle resource usage

Let's see how much `/usr/bin/yes` uses:

```sh
$ rsg --target /usr/bin/yes
2020/02/15 16:14:37 Launching /usr/bin/yes for idle state resource usage assessment
2020/02/15 16:14:37 Trying to determine idle state resource usage (no external traffic)
2020/02/15 16:14:39 Idle state assessment of /usr/bin/yes completed
2020/02/15 16:14:39 Found idle state resource usage. MEMORY: 786kB CPU: 986ms (user)/9ms (sys)
{
 "idle": {
  "memory_in_bytes": 786432,
  "cpuuser_in_usec": 986316,
  "cpusys_in_usec": 9140
 },
 "peak": {
  "memory_in_bytes": 0,
  "cpuuser_in_usec": 0,
  "cpusys_in_usec": 0
 }
}
```

### Assessing the peak resource usage

Now, imagine you have an application called `test` (as shown in [test/](https://github.com/mhausenblas/right-size-guide/blob/master/test/main.go)). 
It's a binary executable that exposes an HTTP API on `:8080/ping`. This is how 
you can use `rsg` to figure out how much memory and CPU this app server requires:

```sh
$ rsg --target test/test --api-path /ping --api-port 8080
2020/02/15 16:20:24 Launching test/test for idle state resource usage assessment
2020/02/15 16:20:24 Trying to determine idle state resource usage (no external traffic)
2020/02/15 16:20:26 Idle state assessment of test/test completed
2020/02/15 16:20:26 Found idle state resource usage. MEMORY: 7757kB CPU: 8ms (user)/11ms (sys)
2020/02/15 16:20:26 Launching test/test for peak state resource usage assessment
2020/02/15 16:20:26 Trying to determine peak state resource usage using 127.0.0.1:8080/ping
2020/02/15 16:20:27 Starting to hammer the endpoint http://127.0.0.1:8080/ping every 10ms
2020/02/15 16:20:36 Peak state assessment of test/test completed
2020/02/15 16:20:36 Found peak state resource usage. MEMORY: 20209kB CPU: 191ms (user)/179ms (sys)
{
 "idle": {
  "memory_in_bytes": 7757824,
  "cpuuser_in_usec": 8634,
  "cpusys_in_usec": 11988
 },
 "peak": {
  "memory_in_bytes": 20209664,
  "cpuuser_in_usec": 191918,
  "cpusys_in_usec": 179626
 }
}
```

You can also specify a file to export the findings to explicitly and suppress 
the status updates, like so:

```sh
$ rsg --target test/test \
      --api-path /ping --api-port 8080 \
      --export-findings ./results.json 2>/dev/null
```

Then, you could pull out specific results, say, using `jq`:

```sh
$ cat results.json | jq .peak.cpuuser_in_usec
76174
```

### Emit OpenMetrics

By default, `rsg` will output the findings in JSON, however it's super easy to
emit OpenMetrics, like so:

```sh
$ rsg --target test/test \
      --api-path /ping --api-port 8080 \
      --output openmetrics
2020/02/15 16:46:01 Launching test/test for idle state resource usage assessment
2020/02/15 16:46:01 Trying to determine idle state resource usage (no external traffic)
2020/02/15 16:46:03 Idle state assessment of test/test completed
2020/02/15 16:46:03 Found idle state resource usage. MEMORY: 7831kB CPU: 8ms (user)/8ms (sys)
2020/02/15 16:46:03 Launching test/test for peak state resource usage assessment
2020/02/15 16:46:03 Trying to determine peak state resource usage using 127.0.0.1:8080/ping
2020/02/15 16:46:04 Starting to hammer the endpoint http://127.0.0.1:8080/ping every 10ms
2020/02/15 16:46:13 Peak state assessment of test/test completed
2020/02/15 16:46:13 Found peak state resource usage. MEMORY: 20168kB CPU: 180ms (user)/168ms (sys)
# HELP idle_memory The idle state memory consumption
# TYPE idle_memory gauge
idle_memory{unit="kB",target="test/test"} 7831552
# HELP idle_cpu_user The idle state CPU consumption in user land
# TYPE idle_cpu_user gauge
idle_cpu_user{unit="microsec",target="test/test"} 8079
# HELP idle_cpu_sys The idle state CPU consumption in the kernel
# TYPE idle_cpu_sys gauge
idle_cpu_sys{unit="microsec",target="test/test"} 8152
# HELP peak_memory The peak state memory consumption
# TYPE peak_memory gauge
peak_memory{unit="kB",target="test/test"} 20168704
# HELP peak_cpu_user The peak state CPU consumption in user land
# TYPE peak_cpu_user gauge
peak_cpu_user{unit="microsec",target="test/test"} 180770
# HELP peak_cpu_sys The peak state CPU consumption in the kernel
# TYPE peak_cpu_sys gauge
peak_cpu_sys{unit="microsec",target="test/test"} 168892
```