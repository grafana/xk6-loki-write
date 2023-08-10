import {sleep, check} from 'k6';
import loki from 'k6/x/ngloki';
import exec from 'k6/execution';

const CONFIG = {
  url: `http://localhost:3100`,
  randSeed: 65,
  addVuAsTenantID: false,
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
        churningLabels: {"pod": 100}, // add a churning label, value will be replaced with a number every 100 ticks
        streams: 8,
        linesPerSec: 20000,
        randomLineSizeMin: 100,
        randomLineSizeMax: 1000,
        // The logType is the default "logfmt"
    },
    {
        staticLabels: {"k6test": "true", "namespace": "loki-prod-002", "container": "distributor"},
        bytesPerSec: 5000,
        logType: "common_log",
    },
    {
        staticLabels: {"k6test": "true", "namespace": "loki-prod-003", "container": "ingester-zone-a-11"},
        linesPerSec: 7500,
        maxLineSize: 100,
        logType: "apache_combined",
        tenantID: "465"
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
}


export function teardown(date) {
  loki.Stop();
}