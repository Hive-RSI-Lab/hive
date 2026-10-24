# Gate gate-1792807756: KEPT

better than the current Hive with no regression: benchmark passes 2/2, the control 0/2

- candidate fa42508c0, control 6e45aaa59, sandbox bare, 48s
- gain: benchmark passes 2/2, the control 0/2
- gain: recurse passes 2/2, the control 0/2

## Candidate

| probe | rep | pass | seconds | metrics | detail |
|---|---|---|---|---|---|
| daemon | 1 | true | 0.5 |  | hive 0.0.0-dev |
| org | 1 | true | 0.0 |  | gate-1792807756-candidate |
| benchmark | 1 | true | 18.3 | score=2 asks_finished=1 seconds=16.06 cost_usd=0 | {"asks":1,"asks_finished":1,"cost_usd":0,"score":2,"seconds":16.059536931} |
| recurse | 1 | true | 1.0 |  | nested gate gate-1792807782: not better than the current Hive: nothing moved beyond the noise |
| daemon | 2 | true | 0.5 |  | hive 0.0.0-dev |
| org | 2 | true | 0.0 |  | gate-1792807756-candidate |
| benchmark | 2 | true | 18.4 | score=2 asks_finished=1 seconds=16.08 cost_usd=0 | {"asks":1,"asks_finished":1,"cost_usd":0,"score":2,"seconds":16.084135924} |
| recurse | 2 | true | 1.0 |  | nested gate gate-1792807803: not better than the current Hive: nothing moved beyond the noise |

## Control

| probe | rep | pass | seconds | metrics | detail |
|---|---|---|---|---|---|
| daemon | 1 | true | 0.5 |  | hive 0.0.0-dev |
| org | 1 | true | 0.0 |  | gate-1792807756-control |
| benchmark | 1 | false | 0.7 |  | …ting: warning: no tests to run PASS ok  	github.com/ihsan-sa/hive-standalone-build/internal/e2e	0.424s [no tests to run] |
| recurse | 1 | false | 0.0 |  | …ish their turn)   hive uninstall           remove autostart and the binary's record; leaves your folder   hive --version |
| daemon | 2 | true | 0.5 |  | hive 0.0.0-dev |
| org | 2 | true | 0.0 |  | gate-1792807756-control |
| benchmark | 2 | false | 0.7 |  | …ting: warning: no tests to run PASS ok  	github.com/ihsan-sa/hive-standalone-build/internal/e2e	0.399s [no tests to run] |
| recurse | 2 | false | 0.0 |  | …ish their turn)   hive uninstall           remove autostart and the binary's record; leaves your folder   hive --version |

Org repositories made: 2, removed: 2, left: 0
