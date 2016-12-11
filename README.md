# bub

`bench-hub` ⇒ `bhub` ⇒ `bub` a cli tool for all your Bench related needs.

## Warning / Disclaimer

Everything here is experimental and the commands set is subject to, and will,
change. Also, when listing the manifests the list is also incomplete, until each
project is properly update.

![Our Lord Savior, Lil Bub!](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Lil_Bub_2013_%28crop_for_thumb%29.jpg/440px-Lil_Bub_2013_%28crop_for_thumb%29.jpg)

## Installation

### From Brew (macOS only)
    $ brew install benchlabs/tools/bub

### From source

Install prereqs below.

    $ make install

## Usage

To be expanded

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
    $ brew install golang # tested with 1.7.3 must fix version in future.
    $ go get github.com/constabulary/gb/... # fix version when required.

## Build

    $ make deps
    $ make
    $ bin/bub<your-platform>

FYI: If you are using oh-my-zsh with the git plugin, `gb` gets aliases to `git branch`. You
can always call `gb` directly with `\gb` or use `unalias gb`.

## Dependency management

    $ gb vendor fetch # to add stuff.
    $ gb vendor # for more options.
