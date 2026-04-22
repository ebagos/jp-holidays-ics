# jp-holidays-ics Project Overview

## Purpose
日本の祝日カレンダー（ICSファイル）を自動生成するGoツール。
内閣府が公開しているCSVデータ（https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv）を取得し、
iCalendar形式（.ics）に変換して `public/japanese-holidays.ics` に出力する。

## Tech Stack
- Language: Go 1.26.2
- External dependencies: なし（標準ライブラリのみ）
- VCS: git + jj (Jujutsu) を併用
- Tool management: mise (mise.toml)

## Repository Structure
```
jp-holidays-ics/
├── cmd/
│   └── generate-ics/
│       └── main.go       # エントリポイント（唯一のGoファイル）
├── .github/
│   └── workflows/
│       └── build-holidays.yml  # GitHub Actions（現在は空）
├── go.mod
├── mise.toml             # jj="latest" のみ定義
└── LICENSE
```

## Key Constants (main.go)
- `sourceURL`: 内閣府の祝日CSVのURL
- `outputPath`: `public/japanese-holidays.ics`
- `calendarID`: `jp-holidays-cao-example`
- `calendarName`: `日本の祝日`
