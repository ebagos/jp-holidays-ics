# jp-holidays-ics

内閣府が公開している[祝日CSVデータ](https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv)を取得し、iCalendar形式（`.ics`）のカレンダーファイルを自動生成するツールです。

## 概要

- **データソース**: 内閣府「国民の祝日について」公開CSVデータ
- **出力形式**: iCalendar (RFC 5545) 準拠の `.ics` ファイル
- **更新**: GitHub Actions により定期的に自動更新

## カレンダーの購読

生成済みの `.ics` ファイルを任意のカレンダーアプリで購読できます。

| アプリ | 手順 |
|--------|------|
| Google カレンダー | 「他のカレンダーを追加」→「URL で追加」からICSのURLを入力 |
| Apple カレンダー | ファイルメニュー →「カレンダーの購読」からURLを入力 |
| Outlook | 「カレンダーの追加」→「インターネットから」からURLを入力 |

## ローカルでの実行

### 前提条件

- Go 1.22 以上

### ICSファイルの生成

```bash
go run ./cmd/generate-ics/
```

実行すると `public/japanese-holidays.ics` が生成（または更新）されます。

### ビルド

```bash
go build ./cmd/generate-ics/
./generate-ics
```

## 技術的な詳細

- 内閣府CSVはShift-JISエンコードのため、実行時にUTF-8へ変換します
- 各イベントのUIDは「日付 + 祝日名 + カレンダーID」のSHA-256ハッシュから生成されるため、データが変わらない限り同一のUIDが維持されます
- 外部ライブラリへの依存は最小限（[`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text) のみ）

## データについて

祝日データは内閣府が提供するデータをそのまま利用しています。データの正確性については内閣府の公開情報に依拠します。

- [内閣府「国民の祝日について」](https://www8.cao.go.jp/chosei/shukujitsu/gaiyou.html)

## コントリビューション

バグ報告や改善提案は [Issues](../../issues) へお気軽にどうぞ。Pull Request も歓迎します。

1. このリポジトリをフォーク
2. フィーチャーブランチを作成 (`git checkout -b feature/your-feature`)
3. 変更をコミット
4. ブランチをプッシュ
5. Pull Request を作成

## ライセンス

[MIT License](LICENSE) © 2026 ebagos
