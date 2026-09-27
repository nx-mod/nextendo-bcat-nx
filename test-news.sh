#!/usr/bin/env sh
# Example: serve Nextendo news to the console.
# Start the server (./run.sh) then run this. It fetches the news feed the way the
# console's News/topics sync would, and prints the item.
set -e
BASE="${1:-http://localhost:8470}"
echo "== /news =="
curl -s "$BASE/news" ; echo
echo "== /topics (what a BCAT topics sync sees) =="
curl -s "$BASE/topics" ; echo
echo
echo "Edit news.json (or set BCAT_NEWS) to change the headline; the default is"
echo "\"nextendo news test\"."
