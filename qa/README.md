# QA Scenarios

This directory hosts automated test scenarios used to validate dispatch behaviour under pre-production conditions.

## Running

Execute all QA scenarios using:

```bash
./qa/run_scenarios.sh
```

The script runs the Go tests located in `qa/scenarios` which load YAML definitions from `qa/scenarios/*.yaml`. Results are summarised in the console and metrics can be inspected in `qa/results`.

## Adding New Scenarios

1. Create a new YAML file under `qa/scenarios/` following the existing format.
2. Define vehicles, signals and expected acknowledgments.
3. Optionally specify `fail_vehicles`, `ack_fail_after` or `disconnect_after` to simulate edge cases.
4. Add an entry in `qa/results/expected_outputs.md` describing the expected behaviour.

Tests automatically discover new files and validate them.
