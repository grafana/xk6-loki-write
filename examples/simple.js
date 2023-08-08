import {sleep, check} from 'k6';
import loki from 'k6/x/ngloki';
import exec from 'k6/execution';

const CONFIG = {
  url: `http://localhost:3100`,
  randSeed: 65,
};
loki.CreateClient(CONFIG);

/**
 * Define test scenario
 */
export const options = {
  vus: 3,
  iterations: 180,
};

export function setup() {
  return {"vuSpecs": [
    {
        staticLabels: {"k6test": "true","namespace": "loki-prod-001", "source": "kafka"},
        lines: 20000,
        bytes: 200,   // TODO: not used yet
        frequency: 5, // TODO: based on state.GetScenarioVUIter() module vus : every 5 ticks/seconds
    },
    {
        staticLabels: {"k6test": "true", "namespace": "loki-prod-002", "container": "distributor"},
        lines: 5000,
        bytes: 5000,
        frequency: 1,
    },
    {
        staticLabels: {"k6test": "true", "namespace": "loki-prod-003", "container": "ingester-zone-a-11"},
        lines: 7500,
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
  let currentVU = exec.vu.idInTest - 1;
  let vuParams = data["vuSpecs"][currentVU];
  // console.log('vuParams: ' + JSON.stringify(vuParams));

  // Write logs
  loki.Tick(vuParams);

  // Wait before next iteration, maybe put this in Tick function
  sleep(1);
}


export function teardown(date) {
  loki.Stop();
}