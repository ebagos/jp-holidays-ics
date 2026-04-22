# Code Style & Conventions

## Go Conventions
- 標準的なGoのコーディングスタイルに従う
- `gofmt` でフォーマットを統一
- コメントは日本語・英語が混在（CSV処理部は日本語コメントあり）
- エラーハンドリングは `exitf()` ヘルパー関数でstderrに出力してexit(1)

## Naming
- 関数名: camelCase（Go標準）
- 定数: camelCase（Go標準、exportしない定数は小文字始まり）
- 型名: PascalCase（例: `Holiday` struct）

## Structure
- 現在は単一ファイル構成（`cmd/generate-ics/main.go`）
- パッケージ: `main`
- 外部ライブラリは使用しない方針（標準ライブラリのみ）

## ICS生成
- RFC 5545に基づくiCalendar形式
- 終日イベントとして登録（DTSTART/DTEND に VALUE=DATE）
- UIDはSHA256ハッシュで生成（日付+タイトル+calendarIDから）
- CRLF改行（\r\n）
