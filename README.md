# Worklog

## Install

`go get github.com/bborbe/worklog`

## Usage

Display commits

```
worklog \
-author "Benjamin Borbe" \
-dir /path/to/git/repo
```

Display multiple commits

```
worklog \
-days 10 \
-author "Benjamin Borbe" \
-dir ~/git_repo_a,~/git_repo_b
```

Sort output

```
worklog \
-days 10 \
-author "Benjamin Borbe" \
-dir ~/git_repo_a,~/git_repo_b | sort
```

## Scripts search for git repos and print worklog

```
#!/bin/bash

function join_by { local IFS="$1"; shift; echo "$*"; }

list=$(find ~/Documents/workspaces/sm-* $GO/src/bitbucket.apps.seibert-media.net -name .git -type d -prune -exec dirname {} \;)

worklog -days 10 -author "Benjamin Borbe" -dir $(join_by , $list) | sort | uniq
```
