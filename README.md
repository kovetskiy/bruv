# bruv

bruv shows difference between branches of remote repositories in
human-readable format.

```
git@github.com:kovetskiy/do     dev is same as master
git@github.com:kovetskiy/what   compared to master, dev is 8 commits behind
git@github.com:kovetskiy/thou   compared to master, dev is 5 commits ahead
git@github.com:kovetskiy/wilt   compared to master, dev is 6 commits ahead and 3 commits behind
```

## Usage

```
Usage:
  bruv [options] <src> <dst> <url>...
  bruv [options] <src> <dst> -i
  bruv [options] <src> <dst> -f <path>
  bruv -h | --help
  bruv --version
Options:
  -i --stdin        Use stdin as list of repositories.
  -f --file <path>  Use specified file as list of repositories.
  -c --cache <dir>  Use this directory for cache.
                     [default: $HOME/.cache/bruv/]
  -j --json         Output in JSON.
  -h --help         Show this screen.
  --version         Show version.
```

- `<src>` means branch that will be used as source of comparison.
Occasionally it is `master`.
- `<dst>` means branch that will be used as destination of comparison.
Occasionally it is `dev`.
- `<url>` URL of repository.

You can specify `-i` or `--stdin` flat and specify list of URLs as standard
input (or pipe with file).

## Installation

```
go get github.com/kovetskiy/bruv
```

### Example

```
 $ bruv master plugins git@github.com:reconquest/gunter
git@github.com:reconquest/gunter        compared to master, plugins is 1 commits ahead and 2 commits behind
```
