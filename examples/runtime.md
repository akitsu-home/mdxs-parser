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

[part1](./runtime/part1.md)

### Platforms

- linux
- amd64

### Settings

| Key      | Value |
| -------- | ----- |
| env      | prod  |
| replicas | 2     |
