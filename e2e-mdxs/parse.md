# parse service as json

## command

- go
- run
- ./cmd/mdxs-parser
- parse
- examples/service.md
- --json

## stdout contains

- metadata
- Service
- Runtime Details

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

## stdout contains

- ## Runtime Details
- Visit [project page](https://example.com/project).

## stdout not contains

- [Runtime details](runtime.md#runtime-details)

## stderr not contains

- panic
