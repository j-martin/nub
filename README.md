# bub

`bench-hub` -> `bhub` -> `bub` a cli tool for all your Bench related needs.

## Warning / Disclaimer

Everything here is experimental and the commands set is subject to, and will,
change. Also, when listing the manifests the list is also incomplete, until each
project is properly update.

![](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Lil_Bub_2013_%28crop_for_thumb%29.jpg/440px-Lil_Bub_2013_%28crop_for_thumb%29.jpg)

## Installation

### From Brew (macOS only)
    $ brew install benchlabs/tools/bub

To use ssh key for authentication instead of the default https+basic auth and
avoid requirements for 2FA, add this to your `~/.gitconfig`:

```
[url "git@github.com:"]
  insteadOf = "https://github.com/"
```

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

To be expanded, when in doubt, `-h` with any action/sub-actions should give you
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
