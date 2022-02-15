name: Create Release

on:
  push:
    tags:
      - "v*.*.*"

permissions:
  contents: write # For creating releases

jobs:
  call-create-release:
    uses: aws-controllers-k8s/.github/.github/workflows/reusable-create-release.yaml@main
