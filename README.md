# bub

`bench-hub` -> `bhub` -> `bub` a developer workflow cli tool initially developed
during my time at [bench.co](https://bench.co).

![](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Lil_Bub_2013_%28crop_for_thumb%29.jpg/440px-Lil_Bub_2013_%28crop_for_thumb%29.jpg)

## Warning / Disclaimer

Everything here is experimental and the commands are subject to, and will,
change. Also, when listing the manifests the list is also incomplete, until each
project is properly updated. Also the tool in it's current state largely depends
on the toolchain of my current employer. E.g. Migrating from AWS to GCP. Refer
to [Bench Era](../../tree/bench-era) branch for integration with AWS, CircleCI, Vault, Manifest
and Splunk, etc.


## Installation

### From source

Install prereqs below.

    $ make install

## Setup

To setup bub, you need to have AWS credentials. Run:

    $ bub setup

## Usage

To be expanded, when in doubt, `-h` with any command/sub-command should give you
an idea of what you can do.

    # from any directory
    $ bub # or with --help

    # in a repo
    $ bub gh repo
    $ bub gh issues
    # ...

## Prerequisites

    # macOS to use the open commands (you can symlink xdg-open to open on Linux)
    $ brew install golang # tested with 1.10 must fix version in future.

## Build

    $ make deps
    $ make
    $ bin/bub<your-platform>

    $ make install
    $ make dev
