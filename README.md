# xk6-loki-write
Loki write plugin for k6.

 This uses each K6 virtual user (VU) as a substitute for a Grafana agent. Adding more VUs means adding more agent.
 Each VUs can specify the nr of lines or bytes that will be sent per second.

## Build instructions

1. Install xk6 via these [installation instructions](https://github.com/grafana/xk6#install-xk6)
2. Run `xk6 build --with xk6-loki-write=.`
3. Run an example test using `./xk6 run example/simple.js`

## Configuration parameters

Before running a test update the following in the test script:

1. Update the CONFIG struct:
   * Set the url field in the CONFIG struct to the Loki you will be sending logs to.
   * The randSeed field is the random value used to initialize the fake log data library. By keeping it the same the same logs wil be generated.
   * Set addVuAsTenantID to true to send a `X-Scope-OrgID` header set to the current VU ID. If set to false no such header is sent.
2. Update the `vuSpecs` array in the `setup` function:

   For each VU a new VU specification json object should created and added to the vuSpecs array. Each VU specification looks like this:

   ```
   {
       staticLabels: {"k6test": "true","namespace": "loki-prod-001", "source": "kafka"},
       churningLabels: {"pod": 100}, // add a churning label, value will be replaced with a number every 100 ticks
       linesPerSec: 20000,
       // bytesPerSec: 200000,
       maxLineSize: 100,
       logType: "apache_combined",
   },
   ```

   | Field             | Description                                                                                                                                            | Default   |
   | :---------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- | :-------- |
   | staticLabels      | a map where the string keys are the static label name and the string values the label values                                                           | empty     |
   | churningLabels    | a map where the string keys are the churning label name and the int value is the quotient of the current run of the VU devideb by the value in the map | empty     |
   | streams           | add a label named stream with as value a random number between [0, streams) for each line, not enabled if 0                                            | 0         |
   | linesPerSec       | the number of lines to send per second, mutually exclusive with bytesPerSec, not enabled if 0                                                          | 0         |
   | bytesPerSec       | the number of bytes to send per second, mutually exclusive with linesPerSec, not enabled if 0                                                          | 0         |
   | maxLineSize       | maximum length of a line, if short this will be the likely length of the line, not enabled if 0, this can result in invalid log line for the logType   | 0         |
   | randomLineSizeMin | the line size will be randomly chosen between this and randomLineSizeMax, not used if maxLineSize is set, not enabled if 0, shorter lines remain as is | 0         |
   | randomLineSizeMax | the line size will be randomly chosen between randomLineSizeMax and this, not used if maxLineSize is set, not enabled if 0                             | 0         |
   | logType           | the type of log lines to send, must be one of "apache_common", "apache_combined", "apache_error", "rfc3164", "rfc5424", "common_log", "json", "logfmt" | "logfmt"  |
   | tenantID          | overwrites the tenantID (if any)                                                                                                                       | ""        |

   At least one of linesPerSec or bytesPerSec has to be given.


## Running in Kubernetes

1. Install the k6 operator as mentioned in https://k6.io/blog/running-distributed-tests-on-k8s/

2. Build a new k8s image to be used by the k6 operator:

   ```
   docker build --platform linux/amd64 -f k8s/Dockerfile.k8s -t k6-extended:local .
   ```

   Upload the resulting image to a registry where it can be downloaded from Kubernetes.

3. Create a configmap containing the test script:

   ```
   kubectl create configmap my-test -n k6 --from-file examples/simple.js
   ```

4. Create a K6 CR referencing the new image. For example :

   ```
   apiVersion: k6.io/v1alpha1
   kind: K6
   metadata:
     name: k6-sample
   spec:
     parallelism: 4
     script:
       configMap:
         name: my-test
         file: simple.js
     runner:
     image: <registry>/k6-extended:local
   ```

   This will be picked up the k6 operator and run as jobs whose name starts with `k6-sample`. This example can be found in the k8s directory.
