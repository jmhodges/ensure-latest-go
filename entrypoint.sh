#!/bin/bash

set -euo pipefail

GO_VERSION=$(latest_go_ensurer $@)
echo "GO_VERSION though? ${GO_VERSION}"
echo ::set-output name=go_version::$GO_VERSION
