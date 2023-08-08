import {sleep, check} from 'k6';
import loki from 'k6/x/ngloki';
import exec from 'k6/execution';

/**
 * URL used for push and query requests
 * Path is automatically appended by the client
 * @constant {string}
 */
// const BASE_URL = `http://localhost:3100`;

/**
 * Instantiate config and Loki client
 * Define test scenario
 */
// const conf = new loki.Config(BASE_URL);
// const client = new loki.Client(conf);

/**
 * Define test scenario
 */
export const options = {
  vus: 2,
  iterations: 6,
};

export function setup() {
  return {"vuSpecs": [
    {
        "staticLabels": {"namespace": "loki-prod-001", "source": "kafka"},
        lines: 100,
        bytes: 200,
        frequency: 5, // Based on state.GetScenarioVUIter() module vus : every 5 ticks/seconds
    },
    {
        "staticLabels": {"namespace": "loki-prod-001", "container": "distributor"},
        lines: 1000,
        bytes: 5000,
        frequency: 1,
    },
  ]};
}

/**
 * "main" function for each VU iteration
 */
export default (data) => {
  // Get the VU number
  let currentVU = exec.vu.idInTest - 1
  let vuParams = data["vuSpecs"][currentVU]
  // console.log('vuParams: ' + JSON.stringify(vuParams))

  // Run the eventloops
  loki.Tick(vuParams)

  // Wait before next iteration, maybe put this in Tick function
  sleep(1);
}