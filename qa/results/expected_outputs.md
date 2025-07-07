# Expected Outputs

This file documents the expected outcome for each predefined QA scenario. The Go tests in `qa/scenarios` assert these expectations automatically.

- `nominal_high_ack.yaml` – All vehicles acknowledge, metrics report two acknowledgments.
- `ack_loss_mid_dispatch.yaml` – Second dispatch loses an acknowledgment from `v2`.
- `vehicle_disconnect.yaml` – Vehicle `v3` disconnects after the first signal; subsequent dispatches exclude it.
- `mixed_fleet_partial_compliance.yaml` – Only compliant vehicles (`v1` and `v4`) acknowledge.
- `zero_power_rapid.yaml` – Zero‐power dispatch is ignored but still acknowledged, followed by two rapid signals.
