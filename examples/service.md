---
title: Service Example
owner: platform-team
tags:
  - api
  - worker
---

# Service

API と Worker を含むサービス構成です。

[Runtime details](runtime.md#runtime-details)

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
