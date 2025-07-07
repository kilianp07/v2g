#!/bin/bash
set -e

echo "Running QA scenarios..."
go test ./qa/scenarios -run TestScenario -v
