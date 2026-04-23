---
title: Runtime Example
environment: production
---

# Runtime

## Runtime Details

本番環境を想定した設定です。

```bash
./mdxs-parser parse examples/service.md --json
```

### Part 1

#### hello

testtesttest

##### test

###### hello

testtesttest

###### test

ん？



### Script Import

[python](./runtime/hello.py)

### Platforms

- linux
- amd64

### Settings

| Key      | Value |
| -------- | ----- |
| env      | prod  |
| replicas | 2     |

