go-diff-mod
===========

Examples
--------

- https://github.com/slackhq/nebula/pull/99

Directions
----------

1. Run `go list -m -json all | jq -s . >before.json`
2. Do your package upgrades / installs
3. Run `go list -m -json all | jq -s . >after.json`
4. Run `go-diff-mod before.json after.json`
