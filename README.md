# memo

ローカルのJSONファイルにMarkdown形式のメモを保存する、個人用のGo製CLIツールです。v0.2では、タイトルと本文の追加・編集、一覧・詳細表示・検索・削除、保存先の切り替えができます。

## セットアップ

Go 1.26.4以降が必要です（[go.mod](go.mod)を参照）。リポジトリを取得してビルドします。

```bash
git clone https://github.com/co191194/memo-app.git
cd memo-app
go build -o memo ./cmd/memo
```

以下の例では、ビルドした `./memo` を使います。`memo` を実行したい場合は、バイナリを `PATH` の通った場所へ置いてください。リポジトリには `go.mod` があるため、`go mod init` は不要です。

```bash
./memo add "Goのメモ" --body "interfaceについて調べる"
./memo show 1
./memo search interface
./memo edit 1 --body "interfaceと実装について調べる"
./memo list
```

初期設定では `~/.memo/memos.json` に保存します。ファイルがまだない場合はメモ0件として扱い、初回の保存時に親ディレクトリとファイルを作成します。既存のメモを削除したくない場合は、以下の `MEMO_PATH` で別の保存先を指定して試してください。

## コマンド一覧

| コマンド | 説明 |
|---|---|
| `./memo add <title> [--body <body>]` | メモを追加する |
| `./memo edit <id> [--title <title>] [--body <body>]` | 指定したメモを編集する（少なくとも一方のオプションが必要） |
| `./memo list` | メモのID・タイトル・作成日を一覧表示する |
| `./memo show <id>` | タイトル・本文・作成日時・更新日時などを表示する |
| `./memo search <keyword>` | タイトルまたは本文から検索する |
| `./memo delete <id>` | 指定したメモを削除する（確認なし） |
| `./memo help` | コマンド一覧を表示する |

個別の使い方は `./memo <command> --help` でも確認できます。IDは1以上の整数です。

### 追加・表示・検索

```bash
./memo add "Goのメモ" --body "interfaceについて調べる"
./memo add "タイトルのみのメモ"
./memo add "複数行のメモ" --body $'1行目\n2行目'
./memo list
./memo show 1
./memo search interface
```

`--body` を省略したときの本文は空文字です。本文の改行もそのまま保存され、`show` で表示できます（上の `$'…'` はBashの記法です）。`search` は大文字・小文字を区別します。追加時の作成日時と更新日時は同じです。`list` と `search` はID・タイトル・作成日を表示し、`show` は本文を含む詳細を表示します。

### 編集・削除

```bash
./memo edit 1 --title "新しいタイトル"
./memo edit 1 --body "更新後の本文"
./memo edit 1 --title "タイトル" --body "本文"
./memo edit 1 --body ""  # 本文を空にする
./memo show 1
./memo delete 1
```

編集では指定していない項目は変更しません。IDと作成日時は維持し、更新日時を編集時刻に変更します（同じ値を指定した場合も更新します）。`--title` の空文字・空白のみの指定、更新項目なし、存在しないIDはエラーになります。`add`、`edit`、`delete` は成功時に出力しません。

## 保存先を変更する（`MEMO_PATH`）

環境変数 `MEMO_PATH` を指定した場合、そのJSONファイルだけを読み書きします。コマンドごとに同じ値を指定するか、シェルで環境変数をエクスポートしてください。

```bash
MEMO_PATH=/tmp/work-memos.json ./memo add "Work"
MEMO_PATH=/tmp/work-memos.json ./memo list

export MEMO_PATH="$HOME/work-memos.json"
./memo list
```

未設定、空文字、空白文字だけの場合は既定の `~/.memo/memos.json` を使います。`~/` で始まるパスはホームディレクトリに展開され、相対パスは実行時のカレントディレクトリを基準とします。`~` 単独や `~user/` は展開されません。空白以外を含むパスの前後の空白は削除しません。v0.2にグローバルな `--data-file` オプションはありません。

## 保存形式と既存データ

メモはJSON配列として保存されます。主なフィールドは `id`、`title`、`body`、`created_at`、`updated_at` です。日時はRFC 3339形式です。

```json
[
  {
    "id": 1,
    "title": "Goのメモ",
    "body": "interfaceについて調べる",
    "created_at": "2026-09-27T18:00:00+09:00",
    "updated_at": "2026-09-27T18:00:00+09:00"
  }
]
```

v0.1.0で保存したJSONデータは移行操作なしでそのまま読み書きできます。通常のアップデートでは既定の保存先を変える必要はありません。`MEMO_PATH` で切り替えたファイルのメモは、既定のファイルとは別に管理されます。不正なJSONや読み書きできない保存先はエラーになり、既存のJSONを上書きしません。

## v0.2の完成条件と確認

- 本文の追加・検索・詳細表示、タイトルと本文の編集、既存の一覧・削除が動作する。
- `MEMO_PATH` で保存先を切り替えられ、v0.1.0のJSONを移行なしで使用できる。
- 主要な正常系・異常系のテストと、Pull RequestのCI（書式、静的解析、race detector付きテスト、ビルド）が成功する。
- v0.2マイルストーンのIssueが完了し、READMEと[v0.2.0リリースノート](docs/releases/v0.2.0.md)を確認したうえでリリースする。

ローカルでの確認:

```bash
go vet ./...
go test -race ./...
go build ./...
```

## 開発の背景

v0.1のMVPでは `add`、`list`、`show`、`search`、`delete` とJSONへの永続化を段階的に実装しました。当時の「これから作る」開発手順や仮のディレクトリ構成は、現行のセットアップ手順・仕様ではありません。過去の状態は[v0.1.0タグ](https://github.com/co191194/memo-app/tree/v0.1.0)、v0.2の要件と受け入れ基準は[開発文書](docs/development/v0.2/requirements.md)と[仕様書](docs/development/v0.2/specification.md)を参照してください。現在の実装は `cmd/memo`、`internal/cli`、`internal/memo`、`internal/storage` に分かれています。

将来的な拡張候補には、タグ機能、SQLite対応、Markdown出力などがあります。`edit`、`MEMO_PATH`、CIはv0.2で対応済みです。

## ライセンス

MIT License
