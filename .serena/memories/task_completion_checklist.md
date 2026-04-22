# Task Completion Checklist

タスク完了時に以下を確認すること：

1. **フォーマット**: `gofmt -l .` でフォーマット崩れがないか確認
2. **vet**: `go vet ./...` でエラーがないか確認
3. **ビルド**: `go build ./cmd/generate-ics/` でビルドが通るか確認
4. **テスト**: `go test ./...` でテストが通るか確認（現時点ではテストファイルなし）
5. **動作確認**: `go run ./cmd/generate-ics/` でICSファイルが正常に生成されるか確認
6. **モジュール整理**: 依存関係を追加・削除した場合は `go mod tidy`
