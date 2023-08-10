import {sleep, check} from 'k6';
import loki from 'k6/x/ngloki';
import exec from 'k6/execution';

const CONFIG = {
  // url: `http://localhost:3100`
  url: `https://5901:glc_eyJvIjoiNDAzODE1IiwibiI6InN0YWNrLTEzNjU2Mi1obC10ZXN0azYiLCJrIjoiTloxTnZQNDFLYm14NDFHczQwbTU0MlNvIiwibSI6eyJyIjoidXMifX0=@logs-prod-us-central1.grafana.net/loki/api/v1/push`,
  randSeed: 65,
  addVuAsTenantID: true,
};
loki.CreateClient(CONFIG);

/**
 * Define test scenario
 */
export const options = {
  vus: 4,
  iterations: 480,
};

export function setup() {
  const small_users = [
    {
        staticLabels: {"k6test": "true", "namespace": "loki-prod-002", "container": "distributor"},
        linesPerSec: 50,
        logType: "common_log",
        tenantID: "2",
    },
    {
        staticLabels: {"k6test": "true", "namespace": "loki-prod-003", "container": "ingester-zone-a-11"},
        churningLabels: {"pod": 80}, // add a churning label, value will be replaced with a number every 100 ticks
        linesPerSec: 100,
        maxLineSize: 80,
        logType: "apache_combined",
        tenantID: "3",
    },
  ];

  const big_user = Array(2).fill(
    {
      staticLabels: {"k6test": "true","namespace": "loki-prod-001", "source": "kafka"},
      churningLabels: {"pod": 100, "subpod": 200}, // add a churning label, value will be replaced with a number every 100 ticks
      streams: 30,
      linesPerSec: 20000,
      randomLineSizeMin: 100,
      randomLineSizeMax: 110,
      tenantID: "1",
    },
  );

  return {"vuSpecs": small_users.concat(big_user)};
}

/**
 * "main" function for each VU iteration
 */
export default (data) => {
  // Get the VU number
  let currentVU = exec.vu.idInTest - 1;
  let vuParams = data["vuSpecs"][currentVU];

  // Write logs
  loki.Tick(vuParams);
}


export function teardown(date) {
  loki.Stop();
}