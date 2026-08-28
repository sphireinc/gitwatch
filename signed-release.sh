#!/bin/bash

git status --short
  VERSION=1.0.6 COMMIT=be36c60 \
    SIGNING_KEY=BA143863056C808A \
    ./scripts/signed-release.sh
