# mdxs-parser

Go で実装されたクロスプラットフォーム CLI アプリケーションの初期テンプレートです。
GitHub Releases を一次配布元として、Homebrew Tap と Scoop Bucket からインストールできる構成を前提にしています。

## 主な機能

- Cobra ベースのサブコマンド構成
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
```

`version` の出力例:

```text
mdxs-parser version 1.2.3
commit: abcdef1
built: 2026-03-20T12:34:56Z
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