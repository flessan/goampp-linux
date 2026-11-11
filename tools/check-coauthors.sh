#!/bin/sh
# Validate an explicitly selected commit range; this is not a project-wide
# contributor policy. Names/emails are supplied by the caller, never inferred.
set -eu
if [ "$#" -ne 4 ]; then
    echo 'usage: check-coauthors.sh BASE HEAD NAME EMAIL' >&2
    exit 2
fi
base=$1 head=$2 name=$3 email=$4
expected="Co-authored-by: $name <$email>"
commits=$(git rev-list "$base..$head")
[ -n "$commits" ] || { echo 'No commits in range' >&2; exit 1; }
for commit in $commits; do
    trailers=$(git show -s --format=%B "$commit" | grep -i '^Co-authored-by:' || :)
    if [ "$trailers" != "$expected" ]; then
        echo "Unexpected co-author trailers in $commit" >&2
        exit 1
    fi
done
