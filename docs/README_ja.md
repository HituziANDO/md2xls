# md2xls

[English README](../README.md)

![md2xls image](../sample/assets/md2xls.png)

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)

MarkdownファイルをExcel（.xlsx）ドキュメントに変換するCLIツールです。

## 概要

md2xlsはMarkdownファイルを読み込み、その構造を解析して、スタイル付きのExcelワークブックを生成します。Markdownで文書を作成し、納品やレビュー用にExcelが求められる組織において、設計書や仕様書などの技術文書をExcelファイルとして共有する必要がある場合に便利です。

見出し（自動採番付き）、テーブル（セル内のリッチテキスト書式）、コードブロック、引用、画像、リンクおよびオートリンク（Excelハイパーリンクとして）、リストなどの文書構造を保持し、各要素に適切なExcelスタイルを適用して出力します。インライン書式（太字、斜体、取り消し線、コードフォント）はExcelリッチテキストとして描画されます。HTMLエンティティとHTMLコメントは自動的に処理されます。

## インストール

### `go install` を使用する場合

```sh
go install github.com/HituziANDO/md2xls@latest
```

### ソースからビルドする場合

```sh
git clone https://github.com/HituziANDO/md2xls.git
cd md2xls
CGO_ENABLED=0 go build -o md2xls .
```

### ビルド済みバイナリ（goreleaser）

```sh
goreleaser build --snapshot --clean
```

Linux、macOS、Windows（amd64/arm64）向けのビルド済みバイナリは[Releases](https://github.com/HituziANDO/md2xls/releases)ページから入手できます。

## 使い方

### プロジェクトの初期化

カレントディレクトリにデフォルト値で `.m2x.yml` 設定ファイルを生成します：

```sh
md2xls init
```

`.m2x.yml` が既に存在する場合、上書きせずにコマンドは即座に終了します。

### 基本的な使い方

`.m2x.yml` 設定ファイルがあるディレクトリで `md2xls` を実行します：

```sh
md2xls
```

設定のデフォルト値（入力: `README.md`、出力: `README.xlsx`）が読み込まれます。

### CLIフラグ

| フラグ | 短縮形 | デフォルト | 説明 |
|------|-----------|---------|-------------|
| `--src` | `-s` | （設定ファイルの値） | 入力Markdownファイルのパス |
| `--dst` | `-d` | （設定ファイルの値） | 出力Excelファイルのパス |
| `--config` | `-c` | `.m2x.yml` | 設定ファイルのパス |
| `--version` | `-v` | | バージョンを表示して終了 |
| `--no-heading-number` | | | 見出しの自動採番を無効化 |

CLIフラグは設定ファイルの対応する値を上書きします。

### 使用例

デフォルト設定で特定のファイルを変換する：

```sh
md2xls -s docs/spec.md -d output/spec.xlsx
```

カスタム設定ファイルを使用する：

```sh
md2xls -c my-config.yml
```

インストール済みのバージョンを確認する：

```sh
md2xls -v
```

## 設定

md2xlsは `.m2x.yml` YAMLファイルで設定します。すべてのフィールドは省略可能です。ファイル自体が存在しない場合はデフォルト値が使用されます。

### 設定例（全項目）

```yaml
src: docs/design.md
dst: output/design.xlsx
text:
  font:
    family: Arial
    size: 11
code:
  font:
    family: Courier New
    size: 10.5
max_num_of_characters_per_line: 100
heading_number: true
sheet_name: Sheet1
table_merge_threshold: 80
heading_font_size:
  h1: 24
  h2: 20
  h3: 16
  h4: 14
  h5: 12
  h6: 11
```

### 設定リファレンス

| キー | 型 | デフォルト | 説明 |
|-----|------|---------|-------------|
| `src` | string | `README.md` | 入力Markdownファイルのパス |
| `dst` | string | `README.xlsx` | 出力Excelファイルのパス |
| `text.font.family` | string | `Meiryo UI` | 見出し、本文、テーブルのフォントファミリー |
| `text.font.size` | float | `11.0` | 本文とテーブルのフォントサイズ（pt） |
| `code.font.family` | string | `Arial` | コードブロックとインラインコードのフォントファミリー |
| `code.font.size` | float | `10.5` | コードブロックとインラインコードのフォントサイズ（pt） |
| `max_num_of_characters_per_line` | int | `120` | 折り返しまでの1行あたりの最大文字数 |
| `heading_number` | bool | `true` | H1〜H4の見出し自動採番を有効化（1., 1.1., 1.1.1., 1.1.1.1.） |
| `sheet_name` | string | `Sheet1` | Excelシート名 |
| `table_merge_threshold` | int | `80` | 幅の広いテーブル列を2つのExcel列に結合するためのバイト閾値 |
| `heading_font_size.h1` | float | `24` | H1見出しのフォントサイズ（pt） |
| `heading_font_size.h2` | float | `20` | H2見出しのフォントサイズ（pt） |
| `heading_font_size.h3` | float | `16` | H3見出しのフォントサイズ（pt） |
| `heading_font_size.h4` | float | `14` | H4見出しのフォントサイズ（pt） |
| `heading_font_size.h5` | float | `12` | H5見出しのフォントサイズ（pt） |
| `heading_font_size.h6` | float | `11` | H6見出しのフォントサイズ（pt） |

**フォントの適用範囲：**

- `text.font` の適用先：H1〜H6見出し、プレーンテキスト、テーブルヘッダー、テーブルセル、リスト項目
- `code.font` の適用先：コードブロックとインラインコード（`` `text` ``）

注記：見出しフォントサイズのデフォルトはH1: 24pt、H2: 20pt、H3: 16pt、H4: 14pt、H5: 12pt、H6: 11ptです。`heading_font_size` 設定でカスタマイズ可能です。`text.font.size` 設定は本文、テーブル、リストにのみ適用されます。

## 対応しているMarkdown機能

### 見出し（H1〜H6）

見出しはデフォルトで階層に基づいた自動採番付きで描画されます。自動採番はH1〜H4に適用され、H5とH6は採番なしで描画されます：

- `# Title` は `1. Title` として描画
- `## Section` は `1.1. Section` として描画
- `### Subsection` は `1.1.1. Subsection` として描画
- `#### Item` は `1.1.1.1. Item` として描画
- `##### SubItem` は `SubItem` として描画（採番なし）
- `###### Detail` は `Detail` として描画（採番なし）

各見出しレベルには、出力において固有の太字スタイルとフォントサイズ（H1: 24pt、H2: 20pt、H3: 16pt、H4: 14pt、H5: 12pt、H6: 11pt斜体）が設定されます。

自動採番を無効にするには、設定ファイルで `heading_number: false` を設定するか、`--no-heading-number` CLIフラグを使用します。無効にすると、見出しは採番なしのプレーンテキストとして描画されます（例：`# Title` は `Title` として描画）。

### テーブル

Markdownテーブルは、罫線付きセル、網掛けヘッダー行、幅の広い列の自動サイズ調整（`table_merge_threshold` バイトを超える列は2つのExcel列に結合）で解析・描画されます。

```markdown
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
```

区切り行による列の配置指定がサポートされています：

- `:---` または `---` は左揃え（デフォルト）
- `:---:` は中央揃え
- `---:` は右揃え

テーブルセル内のインライン書式はExcelリッチテキストとして保持されます：`**太字**`、`*斜体*`、`` `コード` ``、`~~取り消し線~~`、`__underscore__` は、ヘッダーセルとデータセルの両方でそれぞれのスタイルで描画されます。

### コードブロック

フェンスドコードブロックは、結合セル領域（A〜H列）に薄い灰色の背景と設定されたコードフォントで描画されます。

````markdown
```go
fmt.Println("Hello")
```
````

### 引用

引用は、結合セル領域（A〜H列）に斜体フォント、左罫線、薄い灰色の背景で描画されます。

```markdown
> これは引用です。
> 複数行にまたがることができます。
```

`>` で始まる連続した行は1つの引用としてグループ化されます。`>` 行の間に空行があると、別々の引用として扱われます。

### 画像

HTML `<img>` タグとMarkdownの画像構文の両方がサポートされています：

```markdown
![代替テキスト](path/to/image.png)
<img src="path/to/image.png">
```

- **ローカル画像**：Markdownファイルのディレクトリからの相対パスで解決
- **リモート画像**（HTTP/HTTPS）：一時ディレクトリに自動ダウンロード（描画後にクリーンアップ）
- **対応フォーマット**：PNG、JPEG、GIF（SVGは非対応）
- 画像はシートに収まるようにスケーリングされ、品質のためにLanczos3リサンプリングで描画

### リスト

箇条書きリストと番号付きリストがサポートされています（ネストを含む）：

```markdown
- 項目1
  - ネストされた項目
- 項目2

1. 最初
2. 2番目
   1. サブ項目
```

タスクリスト（チェックボックス）もサポートされています：

```markdown
- [ ] 未チェック項目
- [x] チェック済み項目
```

未チェック項目は `☐` で、チェック済み項目は `☑` で描画されます。

### 水平線

水平線（`---`、`***`、`___`）は、細い下罫線として描画されます。

### リンク

Markdownリンクは、青い下線付きテキストのExcelハイパーリンクとして描画されます：

```markdown
[ここをクリック](https://example.com)
```

行に1つ以上のリンクが含まれる場合、最初のリンクのURLがセルのハイパーリンクとして設定されます。表示テキストには書式を除いたリンクテキストが表示されます。

オートリンクもサポートされています：

```markdown
<https://example.com>
```

オートリンクは、URLが表示テキストとリンク先の両方となるExcelハイパーリンクとして描画されます。

### HTMLコメント

HTMLコメントは出力から除去されます：

```markdown
<!-- このコメントはExcelに表示されません -->
テキスト <!-- インラインコメント --> 続きのテキスト
```

行全体がコメントの場合は完全にスキップされます。インラインコメントはテキストから除去されます。

### インライン書式

アスタリスクとアンダースコアの両方の構文が強調に対応しています：太字（`**text**` または `__text__`）、斜体（`*text*` または `_text_`）、取り消し線（`~~text~~`）はセル内で適切な書式のExcelリッチテキストとして描画されます。`***太字斜体***` または `___太字斜体___` の組み合わせもサポートされています。インライン書式はリスト項目にも適用されます。

アンダースコアベースの斜体は `snake_case_names` での誤検出を避けるため、単語境界を使用します。

インラインコード（`` `text` ``）は、リッチテキストモードで設定されたコードフォント（例：`code.font.family`）で描画されます。また、強調解析から保護されます：バッククォート内のアスタリスク（例：`` `*ptr` ``、`` `**kwargs` ``）は太字や斜体のマーカーではなくリテラルテキストとして扱われます。

行にリッチテキスト書式とリンクの両方が含まれる場合、セル全体がハイパーリンク（青い下線付きテキスト）としてスタイルされ、リッチテキストラン内の太字/斜体書式も保持されます。

リッチテキスト書式（太字、斜体、取り消し線、コードフォント）は、`max_num_of_characters_per_line` を超えて行が複数行に分割される場合でも保持されます。

書式マーカーのないプレーンテキストの場合、長い行は複数行に分割されます：

- `[リンクテキスト](url)` はリンクテキストとして表示されます（URLはExcelハイパーリンクとして保持）
- オプションのリンクタイトル（`[text](url "title")`）と画像タイトル（`![alt](url "title")`）は自動的に除去されます

### HTMLエンティティ

HTMLエンティティは、すべてのテキストコンテンツ（見出し、本文、テーブルセル、引用、リスト項目）で自動的にデコードされます：

- `&amp;` は `&` に、`&lt;` は `<` に、`&gt;` は `>` に変換
- 名前付きエンティティ：`&copy;` は © に、`&trade;` は ™ に変換、など
- 数値エンティティ：`&#169;` は © に変換

### テキスト折り返し

`max_num_of_characters_per_line`（デフォルト: 120）を超えるプレーンテキスト行は複数行に分割されます。分割は単語境界を考慮し、単語の途中で分割されないようスペースでの改行を優先します。CJKテキストやスペースのないテキストの場合は、文字ベース（UTF-8ルーンベース）の分割にフォールバックします。

## 未対応の機能

以下のMarkdown機能は現在サポートされていません：

- SVG画像
- ネストされたテーブル
- Setext形式の見出し（`Heading\n=======`; ATX `#` 構文を使用してください）
- 参照リンク（`[text][id]`）と脚注（`[^1]`）
- バックスラッシュエスケープ（`\*斜体にならない\*`）
- インラインHTMLタグ（`<strong>`、`<br>`、`<a>`；`<img>` のみ対応）

## 開発

### 前提条件

- Go 1.24以降

### ビルド

```sh
CGO_ENABLED=0 go build -o md2xls .
```

### テスト

```sh
go test ./...
```

### リリースビルド

```sh
goreleaser build --snapshot --clean
```

`dist/` ディレクトリにlinux/darwin/windows向けのamd64およびarm64バイナリが生成されます。

## ライセンス

MIT
