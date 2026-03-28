---
title: Service Example
owner: platform-team
tags:
  - api
  - worker
---

# Service

API と Worker を含むサービス構成です。

## Runtime Details

本番環境を想定した設定です。

```bash
./mdxs-parser parse examples/service.md --json
```

### Platforms

- linux
- amd64

### Settings

| Key      | Value |
| -------- | ----- |
| env      | prod  |
| replicas | 2     |

Visit [project page](https://example.com/project).

```yaml
name: service
replicas: 2
```

## Components

- api
- worker

## Ports

| Component | Port |
| --------- | ---- |
| api       | 8080 |
| worker    | 9090 |

