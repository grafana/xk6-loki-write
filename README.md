# xk6-ngloki
Next generation Loki plugin for k6

## Build instructions

1. Install xk6 via these [installation instructions](https://github.com/grafana/xk6#install-xk6)
2. Run `xk6 build --with xk6-ngloki=.`
3. Run an example test using `./xk6 run example/simple.js`

## Configuration parameters

Before running a test update the following in the test script:

1. Set the url field in the CONFIG struct to the Loki you will be sending logs to
2. Update the `vuSpecs` array in the `setup` function:

   For each VU a new VU specification json object should created and added to the vuSpecs array. Each VU specification looks like this:

   ```
   {
       staticLabels: {"k6test": "true","namespace": "loki-prod-001", "source": "kafka"},
       churningLabels: {"pod": 100}, // add a churning label, value will be replaced with a number every 100 ticks
       linesPerSec: 20000,
       logType: "apache_combined",
   },
   ```

   | Field          | Description                                                                                                                                            | Default   |
   | :------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- | :-------- |
   | staticLabels   | a map where the string keys are the static label name and the string values the label values                                                           |           |
   | churningLabels | a map where the string keys are the churning label name and the int value is the quotient of the current run of the VU devideb by the value in the map |           |
   | linesPerSec    | the number of lines to send per second                                                                                                                 |           |
   | logType        | the type of log lines to send, must be one of "apache_common", "apache_combined", "apache_error", "rfc3164", "rfc5424", "common_log", "json", "logfmt" | "logfmt"  |

