# bub

bench-hub ⇒ bhub ⇒ `bub` a cli tool for all your bench needs

![Our Lord Savior, Lil Bub!](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Lil_Bub_2013_%28crop_for_thumb%29.jpg/440px-Lil_Bub_2013_%28crop_for_thumb%29.jpg)

## Warning / Disclaimer

Everything here is experimental and the commands set is subject to, and will,
change. Also, when listing the manifests the list is also crap, until each
project is properly update.

## Prerequisites

        $ brew install golang # must fix version in future.
        $ go get github.com/constabulary/gb/... # fix version when required.

## Build

        $ gb vendor restore
        $ gb build
        $ bin/bub

FYI: If you are using oh-my-zsh with the git plugin, `gb` gets aliases to `git branch`. You
can always call `gb` directly with `\gb` or use `unalias gb`.

## Dependency management

        $ gb vendor fetch # to add stuff.
        $ gb vendor # for more options.

## Todos

The one I care the most about is having sub-commands like.

        $ bub open jenkins
        $ bub update <version>

- [ ] A real build deployment workflow through jenkins where users can download
  prebuilt binaries.
- [ ] Integrating -update with jenkins build pipeline.
- [ ] Use sub commands and stabilize the command set (stop use -something).
- [ ] Add more documentation once the commands are stable.
- [ ] Add readme boilerplate.
- [ ] Add unit tests
- [ ] Proper way to enforce go version.
- [ ] Probably will end up writing a Makefile while still using gb.

For more todos `ag -q '//TODO'` there is quite a few.
