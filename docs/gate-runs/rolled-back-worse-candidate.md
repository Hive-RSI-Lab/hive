# Gate gate-1792807804: ROLLED BACK

worse than the current Hive: benchmark seconds: candidate 26.05, control 16.04 (noise ±0.321, n=2/2)

- candidate ce7d3085d, control fa42508c0, sandbox bare, 110s
- regression: benchmark seconds: candidate 26.05, control 16.04 (noise ±0.321, n=2/2)

## Candidate

| probe | rep | pass | seconds | metrics | detail |
|---|---|---|---|---|---|
| daemon | 1 | true | 0.5 |  | hive 0.0.0-dev |
| org | 1 | true | 0.0 |  | gate-1792807804-candidate |
| benchmark | 1 | true | 28.3 | score=2 asks_finished=1 seconds=26.04 cost_usd=0 | {"asks":1,"asks_finished":1,"cost_usd":0,"score":2,"seconds":26.042287752} |
| recurse | 1 | true | 1.0 |  | nested gate gate-1792807863: not better than the current Hive: nothing moved beyond the noise |
| daemon | 2 | true | 0.5 |  | hive 0.0.0-dev |
| org | 2 | true | 0.0 |  | gate-1792807804-candidate |
| benchmark | 2 | true | 28.2 | score=2 asks_finished=1 seconds=26.05 cost_usd=0 | {"asks":1,"asks_finished":1,"cost_usd":0,"score":2,"seconds":26.048798055} |
| recurse | 2 | true | 1.1 |  | nested gate gate-1792807912: not better than the current Hive: nothing moved beyond the noise |

## Control

| probe | rep | pass | seconds | metrics | detail |
|---|---|---|---|---|---|
| daemon | 1 | true | 0.5 |  | hive 0.0.0-dev |
| org | 1 | true | 0.0 |  | gate-1792807804-control |
| benchmark | 1 | true | 18.2 | score=2 asks_finished=1 seconds=16.04 cost_usd=0 | {"asks":1,"asks_finished":1,"cost_usd":0,"score":2,"seconds":16.040958058} |
| recurse | 1 | true | 1.0 |  | nested gate gate-1792807864: not better than the current Hive: nothing moved beyond the noise |
| daemon | 2 | true | 0.5 |  | hive 0.0.0-dev |
| org | 2 | true | 0.0 |  | gate-1792807804-control |
| benchmark | 2 | true | 18.5 | score=2 asks_finished=1 seconds=16.04 cost_usd=0 | {"asks":1,"asks_finished":1,"cost_usd":0,"score":2,"seconds":16.044195423} |
| recurse | 2 | true | 1.1 |  | nested gate gate-1792807913: not better than the current Hive: nothing moved beyond the noise |

Org repositories made: 2, removed: 2, left: 0
