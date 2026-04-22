# Suggested Commands

## Build & Run
```bash
# ビルド
go build ./cmd/generate-ics/

# 実行（ICSファイルを生成・更新）
go run ./cmd/generate-ics/
```

## Test & Lint
```bash
# テスト
go test ./...

# フォーマット確認
gofmt -l .

# フォーマット適用
gofmt -w .

# vet
go vet ./...
```

## Dependency Management
```bash
# モジュール整理
go mod tidy
```

## VCS (jj + git)
```bash
# jjステータス確認
jj status

# gitとしても使用可能
git status
git log --oneline
```
