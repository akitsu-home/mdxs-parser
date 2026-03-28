# parse service as json

## command

- go
- run
- ./cmd/mdxs-parser
- parse
- examples/service.md
- --json

## stdout equals file

expected/service.json

## stderr not contains

- panic

# expand markdown include

## command

- go
- run
- ./cmd/mdxs-parser
- parse
- examples/service.md
- --markdown

## stdout equals file

expected/service.markdown

## stderr not contains

- panic
