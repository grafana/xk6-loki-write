import {sleep, check} from 'k6';
import loki from 'k6/x/ngloki';

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
loki.Setup(
{
    "staticLabels": [{"namespace": "loki-prod-001"}, {"source": "kafka"}],
    "percentOfVus": 10,
    lines: 100,
    bytes: 200,
},
{
    "staticLabels": [{"namespace": "loki-prod-001"}, {"container": "distributor"}],
    "percentOfVus": 90,
    lines: 1000,
    bytes: 5000,
},
);

/**
 * Define test scenario
 */
export const options = {
  vus: 10,
  iterations: 10,
};

/**
 * "main" function for each VU iteration
 */
export default () => {
  // Run the eventloops
  loki.Tick()

  // Wait before next iteration, maybe put this in Tick function
  sleep(1);
}