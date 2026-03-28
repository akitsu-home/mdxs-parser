# mdxs-parser

Markdown を構造化 JSON に変換できる Go 製 CLI ツールです。
見出しをオブジェクト階層として扱い、リスト、コードブロック、テーブル、段落を JSON に変換できます。
また、相対パスの Markdown リンクをインクルードとして展開し、JSON または展開済み Markdown を出力できます。

## 主な機能

- 見出し構造を JSON オブジェクトとして出力
- 段落を `description`、リストを `list`、テーブルをオブジェクト配列として変換
- コードブロックをフェンス直後の言語名キーで格納
- 相対パスの Markdown リンクをインクルードとして展開
- `--json` で JSON 出力、`--markdown` で展開済み Markdown を出力
- `version` コマンドで version / commit / build date を表示
- `completion` コマンドで shell completion を生成
- GoReleaser によるクロスコンパイルと GitHub Releases 配布
- Homebrew Formula / Scoop Manifest のテンプレート同梱

## インストール

### Homebrew (Tap)

```bash
brew tap owner/tap
brew install owner/tap/mdxs-parser
```

### Scoop (Bucket)

```powershell
scoop bucket add owner-bucket https://github.com/owner/scoop-bucket.git
scoop install owner-bucket/mdxs-parser
```

### GitHub Releases から手動インストール

1. GitHub Releases から利用 OS/Arch のアーカイブを取得
2. 展開して `mdxs-parser` (Windows は `mdxs-parser.exe`) を PATH に配置

### インストールスクリプト (Linux/macOS)

最新リリースをインストール:

```bash
curl -fsSL https://raw.githubusercontent.com/akitsu-home/mdxs-parser/main/scripts/install.sh | bash
```

特定バージョンをインストール:

```bash
VERSION=v0.0.1 curl -fsSL https://raw.githubusercontent.com/akitsu-home/mdxs-parser/main/scripts/install.sh | bash
```

インストール先を変更:

```bash
INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/akitsu-home/mdxs-parser/main/scripts/install.sh | bash
```

## 使い方

```bash
mdxs-parser help
mdxs-parser version
mdxs-parser completion bash > /tmp/mdxs-parser.bash
mdxs-parser parse examples/service.md
mdxs-parser parse examples/service.md --markdown
```

`version` の出力例:

```text
mdxs-parser version 1.2.3
commit: abcdef1
built: 2026-03-20T12:34:56Z
```

### `parse` コマンド

```bash
mdxs-parser parse <file> [--json|--markdown]
```

- `--json`: JSON で出力します（デフォルト）
- `--markdown`: 相対パスの Markdown リンクを展開した Markdown を出力します

### パースルール

- ヘッダーはネストしたオブジェクトのキーになります
- 段落は `description` に格納されます
- リストは `list` 配列に格納されます
- コードブロックはフェンスの言語名をキーにした文字列として格納されます
  - 言語名がない場合は `code` キーになります
- テーブルはオブジェクトの配列として格納されます
- ボールドやイタリックなどの文字装飾は無視されます
- Web リンクはそのまま残り、相対パスの Markdown リンクだけが展開対象になります

### サンプル

`examples/` 配下に CLI でそのまま試せるサンプルを用意しています。

```bash
mdxs-parser parse examples/service.md --json
mdxs-parser parse examples/service.md --markdown
```

`examples/service.md` の内容:

````md
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
````

JSON 出力例:

```json
{
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
    "description": "API と Worker を含むサービス構成です。\n\nVisit [project page](https://example.com/project).",
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
    "yaml": "name: service\nreplicas: 2"
  }
}
```

## 開発

### 必要ツール

- Go 1.22+
- make
- goreleaser (snapshot/release-dry-run で使用)
- golangci-lint (任意)

### ローカルコマンド

```bash
make test
make lint
make build
make snapshot
make release-dry-run
```

## リリース

### バージョニング

- Semantic Versioning を採用
- Git tag (`v1.2.3`) でリリース

### GitHub Actions

- CI: push / pull_request で test, vet, lint, build を実行
- Release: `v*` タグ push で GoReleaser を実行し GitHub Releases を作成

### 配布先更新

- Homebrew Tap 更新スクリプト: `scripts/update-homebrew-formula.sh`
- Scoop Bucket 更新スクリプト: `scripts/update-scoop-manifest.sh`
- テンプレート:
	- `packaging/homebrew/mdxs-parser.rb.tmpl`
	- `packaging/scoop/mdxs-parser.json.tmpl`

`owner` などのプレースホルダは実運用の org/repo 名へ置き換えてください。

## 開発者向けドキュメント

詳細な開発手順は [README.developer.md](README.developer.md) を参照してください。
