`bcdl` is a command-line tool that downloads and unzips Bandcamp purchases from download pages.

## The problem

You purchase a bunch of releases on Bandcamp Friday in a single cart—perhaps a band's entire discography—and want to download them all. This typically involves a lot of manual clicking and extracting, especially annoying if you store your music on a server like me.

## Install

```
go install github.com/tylerball/bcdl@latest
```

## Usage

Copy and paste the url from the download page after you have made your purchase.

```
bcdl --format=flac 'https://bandcamp.com/download?cart_id=<id>&sig=<sig>&from=checkout'
```

See `bcdl --help` for more options.
