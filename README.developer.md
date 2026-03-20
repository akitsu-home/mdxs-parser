# Developer README

このドキュメントは、mdxs-parser の開発者向け手順をまとめたものです。

## 1. 目的と前提

- 本プロジェクトは Go 製クロスプラットフォーム CLI の雛形です。
- 配布の正本は GitHub Releases です。
- Homebrew Tap / Scoop Bucket は GitHub Releases の成果物を参照します。

## 2. 必要ツール

- Go 1.22 以上
- make
- git
- goreleaser
- golangci-lint (任意)
- gh (GitHub CLI、任意)

確認コマンド例:

- go version
- make --version
- goreleaser --version

## 3. セットアップ

1. リポジトリを clone
2. 依存を解決
3. テストを実行

実行コマンド:

- go mod tidy
- make test

## 4. ディレクトリ構成

- cmd/mdxs-parser: エントリーポイント
- internal/cli: Cobra コマンド実装
- internal/version: バージョン情報変数
- packaging/homebrew: Formula テンプレート
- packaging/scoop: Scoop manifest テンプレート
- scripts: 配布先更新スクリプト
- .github/workflows: CI / Release ワークフロー

## 5. ローカル開発コマンド

- make test: go test ./...
- make lint: go fmt + go vet + (存在すれば golangci-lint)
- make build: ldflags 付きでローカルビルド
- make snapshot: GoReleaser snapshot ビルド
- make release-dry-run: 公開なしリリース検証
- make check-goreleaser: .goreleaser.yaml の検証

## 6. バージョン情報の埋め込み

バージョン情報は ldflags で埋め込みます。

対象:

- internal/version.Version
- internal/version.Commit
- internal/version.Date

version コマンド表示:

- version
- commit
- built date

## 7. テストと品質基準

最低限の品質ゲート:

- go test ./...
- go vet ./...
- go fmt ./...

推奨:

- golangci-lint run

## 8. リリースフロー

1. main にマージ
2. リリースタグ作成 (例: v1.2.3)
3. tag push をトリガーに Release workflow 実行
4. GoReleaser が GitHub Releases に成果物を公開

生成物:

- darwin/linux/windows + amd64/arm64
- Windows: zip
- Unix: tar.gz
- checksums.txt

## 9. Homebrew / Scoop 更新

本リポジトリはテンプレートと更新スクリプトの雛形を提供します。

- Homebrew: packaging/homebrew/mdxs-parser.rb.tmpl
- Scoop: packaging/scoop/mdxs-parser.json.tmpl

更新スクリプト:

- scripts/update-homebrew-formula.sh
- scripts/update-scoop-manifest.sh

注意:

- owner などのプレースホルダは実運用値へ置換してください。
- CI で自動更新する場合は以下の Secrets を設定してください。
  - HOMEBREW_TAP_TOKEN
  - SCOOP_BUCKET_TOKEN

## 10. GitHub Actions

- CI workflow
  - push / pull_request で test, vet, lint, build
- Release workflow
  - v* タグで GoReleaser を実行
  - 必要に応じて tap/bucket 更新スクリプトを実行

## 11. よくある作業

新しいサブコマンド追加:

1. internal/cli に新規コマンドファイルを作成
2. root command で AddCommand
3. 必要ならテスト追加
4. make test / make lint / make build

## 12. トラブルシュート

- goreleaser コマンドがない
  - goreleaser をインストールして make check-goreleaser を再実行
- version が dev のまま
  - タグ未付与、または ldflags 未指定のローカル実行です
- CI の tap/bucket 更新がスキップされる
  - 対応する token Secret が未設定です
