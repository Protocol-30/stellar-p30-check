#!/usr/bin/env bash
# Source this file from a Bash shell. A normal local Go installation also works.
if [[ -x /workspace/.tools/go1.27.3/go/bin/go ]]; then
  export PATH="/workspace/.tools/go1.27.3/go/bin:$PATH"
  export GOCACHE="${GOCACHE:-/workspace/.cache/go-build}"
  export GOPATH="${GOPATH:-/workspace/.cache/gopath}"
fi
# Never download or switch compilers implicitly when inspecting a repository.
export GOTOOLCHAIN=local
