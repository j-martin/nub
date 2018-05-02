# bub

`bench-hub` -> `bhub` -> `bub` a developer workflow cli tool initially developed
during my time at [bench.co](https://bench.co).

![](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Lil_Bub_2013_%28crop_for_thumb%29.jpg/440px-Lil_Bub_2013_%28crop_for_thumb%29.jpg)

## Warning / Disclaimer

Everything here is experimental and the commands are subject to, and will,
change. Also, when listing the manifests the list is also incomplete, until each
project is properly updated. Also the tool in it's current state largely depends
on the toolchain of my current employer. E.g. Migrating from AWS to GCP.


## Installation

### From source

Install prereqs below.

    $ make install

## Setup

To setup bub, you need to have AWS credentials. Run:

    $ bub setup

First you'll be pompted to enter your AWS credentials in the
`~/.aws/credentials`. Then bub will create the `~/.config/bub/config.yml`. You
don't have to edit it unless you want to add some credentials to get more
features. Adding your Jenkins credentials makes bub super userful.

## Usage

To be expanded, when in doubt, `-h` with any command/sub-command should give you
an idea of what you can do.

    # from any directory
    $ bub # or with --help
    $ bub eb
    $ bub ec2

    # in a repo
    $ bub gh repo
    $ bub gh issues
    # ...

## Prerequisites

    # macOS to use the open commands (you can symlink xdg-open to open on Linux)
    $ brew install golang # tested with 1.8.1 must fix version in future.
    $ brew install graphviz

## Build

    $ make deps
    $ make
    $ bin/bub<your-platform>
