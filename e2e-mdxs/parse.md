# parse service as json

## command

```command
go run ./cmd/mdxs-parser parse examples/service.md --json
```

## stdout equals

```expected
{
  "body": {
    "Service": {
      "Components": [
        "api",
        "worker"
      ],
      "Ports": [
        {
          "Component": "api",
          "Port": "8080"
        },
        {
          "Component": "worker",
          "Port": "9090"
        }
      ],
      "Runtime Details": {
        "Platforms": [
          "linux",
          "amd64"
        ],
        "Settings": [
          {
            "Key": "env",
            "Value": "prod"
          },
          {
            "Key": "replicas",
            "Value": "2"
          }
        ],
        "bash": "./mdxs-parser parse examples/service.md --json",
        "description": "本番環境を想定した設定です。"
      },
      "description": "API と Worker を含むサービス構成です。\n\nVisit [project page](https://example.com/project).",
      "yaml": "name: service\nreplicas: 2"
    }
  },
  "metadata": {
    "owner": "platform-team",
    "tags": [
      "api",
      "worker"
    ],
    "title": "Service Example"
  }
}
```

## stderr not contains

- panic

# expand markdown include

## command

```command
go run ./cmd/mdxs-parser parse examples/service.md --markdown
```

## stdout equals file

```expected
expected/service.markdown
```

## stderr not contains

- panic
