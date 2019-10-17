#!/bin/bash

set -euo pipefail

GO_VERSION=$(latest_go_ensurer $@)
echo ##[set-output name=go_version]$GO_VERSION
