# SeaSS

A linter for a strict subset of CSS.

## Installation

```
go get github.com/mushishi78/seass
```

## Setup

Create a configuration file at the root of you project called 'seass.toml' and give it the files and folders you'd like it to ignore, eg:

```toml
Ignore = [
    ".git",
    "node_modules",
    "dist",
]
```

## Run

From the root of your project:

```
seass
```
