# tokotachi-scaffolds リファレンスマニュアル

本ドキュメントは `tokotachi-scaffolds` リポジトリにテンプレートを作成する際のリファレンスです。

## リポジトリの目的

`tokotachi-scaffolds` は、`tt scaffold` コマンドが使用するテンプレートリポジトリです。テンプレートは GitHub Contents API 経由でダウンロードされ、ユーザーのプロジェクトに適用されます。

---

## リポジトリ全体構造

```
tokotachi-scaffolds/
├── meta.yaml                 # 最小メタデータ（version, default_scaffold, updated_at）
├── catalog.yaml              # インデックス（category → name → shard path）
├── catalog/
│   ├── originals/            # テンプレート元ソース + scaffold 定義
│   │   ├── {org}/
│   │   │   └── {project}/
│   │   │       ├── base/     # ソースファイル群
│   │   │       └── scaffold.yaml  # scaffold 定義（placement 内包）
│   │   └── root/
│   │       └── {project}/
│   └── scaffolds/            # シャーディング出力（templatizer 生成）
│       └── {h[0]}/{h[1]}/{h[2]}/
│           ├── {h[3]}.yaml  # scaffold メタデータ（配列形式）
│           └── {name}.zip   # ZIP アーカイブ
├── features/
│   └── templatizer/          # テンプレート変換ツール
├── scripts/                  # ビルド・テストスクリプト
└── shared/                   # 共有ライブラリ
```

### originals と scaffolds の関係

- `originals/` にはビルド・実行可能な実際のプロジェクトコードと `scaffold.yaml`（scaffold 定義）を配置する
- `templatizer` ツールが `originals/` を変換して `scaffolds/` に ZIP アーカイブとシャーディングメタデータを出力する
- `scaffolds/` は読み取り専用であり、直接編集してはならない（`templatizer` により自動生成される）

---

## ファイル構成

### `meta.yaml`（トップレベル）

templatizer 実行時に自動生成される最小メタデータ。

```yaml
version: "1.0.0"
default_scaffold: "default"
updated_at: "2026-03-10T19:00:00+09:00"
```

| フィールド | 説明 |
|---|---|
| `version` | カタログバージョン |
| `default_scaffold` | パターン未指定時に使われるテンプレート名 |
| `updated_at` | templatizer 実行時のタイムスタンプ（クライアントのキャッシュ判定用） |

### `catalog.yaml`（インデックス、トップレベル）

templatizer 実行時に自動生成されるインデックス。ハッシュ計算不要で scaffold に到達可能。

```yaml
scaffolds:
  root:
    default: "catalog/scaffolds/6/j/v/n.yaml"
  project:
    axsh-go-standard: "catalog/scaffolds/8/w/4/o.yaml"
  feature:
    axsh-go-standard: "catalog/scaffolds/b/i/b/l.yaml"
    axsh-go-kotoshiro-mcp: "catalog/scaffolds/i/4/2/h.yaml"
```

### `scaffold.yaml`（originals 配下の入力ファイル）

各テンプレートの `originals/` ディレクトリ直下に配置する scaffold 定義。配置ルール（placement）を内包する。

```yaml
name: "<テンプレート名>"
category: "<カテゴリ名>"
description: "<説明文>"
depends_on:                       # 依存先 scaffold（任意、配列形式）
  - category: "<依存先カテゴリ>"
    name: "<依存先名>"
original_ref: "catalog/originals/<org>/<dir>"
placement:                        # 配置ルール（内包）
  base_dir: "."
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions:
    gitignore_entries: []
    file_permissions: []
template_params:                  # テンプレートパラメータ（任意）
  - name: "<パラメータ名>"
    description: "<説明>"
    required: true|false
    default: "<デフォルト値>"
    old_value: "<元の値>"          # 変換時の置換元（省略時はdefaultを使用）
```

### 主要フィールド

| フィールド | 説明 |
|---|---|
| `name` | テンプレートの識別名。`tt scaffold <name>` で指定する |
| `category` | テンプレートのカテゴリ（`root`, `project`, `feature` 等） |
| `description` | テンプレートの説明文 |
| `depends_on` | 依存先 scaffold のリスト（配列形式、任意） |
| `original_ref` | テンプレート元ソースのパス（`catalog/originals/` 以下） |
| `placement` | 配置ルール（内包） |
| `template_params` | テンプレートパラメータ定義（変換ルールを含む） |

### パターン解決ルール

| コマンド | 解決ロジック |
|---|---|
| `tt scaffold` | `default_scaffold` の値で `name` を検索 |
| `tt scaffold <name>` | `name` フィールドで完全一致検索 |
| `tt scaffold <category>` | `category` で検索し候補一覧を返す |
| `tt scaffold <category> <name>` | `category` + `name` の両方一致で検索 |

### 前提条件 (`requirements`)

- `directories`: 指定パスのディレクトリがプロジェクトに存在しなければエラー
- `files`: 指定パスのファイルがプロジェクトに存在しなければエラー
- 未充足時は `tt scaffold default` 等を先に実行するようヒントを表示

### 依存関係 (`depends_on`)

`depends_on` は依存先 scaffold を配列形式で指定します。各要素は `category` と `name` フィールドを持ちます。

```yaml
depends_on:
  - category: "root"
    name: "default"
```

| フィールド | 型 | 説明 |
|---|---|---|
| `depends_on[].category` | string | 依存先の `category` |
| `depends_on[].name` | string | 依存先の `name` |

- 省略または空配列の場合は依存なし
- 複数要素を指定することで複数の依存先を定義可能
- 循環依存はエラーとなる
- `tt scaffold` 実行時、依存チェーンをトポロジカルソートで解決し、依存元（ルート）から順に適用する

### シャーディング構造 (`catalog/scaffolds/`)

scaffold 定義はハッシュベースで個別ファイルに分割されます。templatizer 実行により自動生成されます。

**ハッシュ計算:**

```
hash = base36(FNV_1a_32(category + "/" + name) % 1679616)  → 4文字、0パディング
```

**ディレクトリ構造:**

```
catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/
├── {h[3]}.yaml              ← scaffold メタデータ（配列形式）
└── {original-basename}.zip  ← ZIP アーカイブ
```

各階層は最大36エントリ（`0-9a-z`）。例: ハッシュ `a3k9` → `catalog/scaffolds/a/3/k/9.yaml`

**シャーディングファイルのフォーマット:**

```yaml
scaffolds:
  - name: "axsh-go-standard"
    category: "feature"
    template_ref: "catalog/scaffolds/b/i/b/go-standard-feature.zip"
    original_ref: "catalog/originals/axsh/go-standard-feature"
    # ... その他のフィールド
```

- 配列形式（ハッシュ衝突時に複数エントリを格納可能）
- `template_ref` は同ディレクトリの ZIP ファイルを参照
- ZIP ファイル名衝突時は `{name}-{n}.zip`（n=2, 3, ...）で連番
- クライアントは `category` + `name` でフィルタして目的のエントリを取得

### テンプレートパラメータ (`template_params`)

`template_params` はテンプレート変数と変換ルールを統合的に定義します。

| フィールド | 説明 |
|---|---|
| `name` | パラメータ名（`module_path`, `program_name` 等） |
| `description` | パラメータの説明 |
| `required` | 必須かどうか（`true` の場合、CLI で入力を促す） |
| `default` | デフォルト値 |
| `old_value` | テンプレート変換時の置換元値（省略時は `default` が使用される） |
| `value_spec` | パラメータ値のバリデーション仕様（省略可、下記参照） |

#### 値仕様 (`value_spec`)

パラメータに入力される値の型やバリデーションルールを定義します。

| フィールド | 型 | 説明 |
|---|---|---|
| `type` | string | パラメータの型。`string`（文字列）または `number`（整数）。デフォルト: `string` |
| `length` | object | 長さ制約 |
| `length.max_bytes` | int | 最大バイト数 |
| `length.max_chars` | int | 最大文字数（ルーン数） |
| `length.max_digits` | int | 最大桁数（`number` 型の場合） |
| `format` | object | フォーマット制約 |
| `format.pattern` | string | Go `regexp` 互換の正規表現パターン |
| `range` | object | 範囲制約（JSONSchema スタイル） |
| `range.minimum` | float64 | 最小値（包含） |
| `range.maximum` | float64 | 最大値（包含） |
| `range.exclusive_minimum` | float64 | 最小値（排他） |
| `range.exclusive_maximum` | float64 | 最大値（排他） |
| `enum` | []string | 許可される値のリスト |

**記述例:**

```yaml
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myfunction"
    value_spec:
      type: string
      length:
        max_bytes: 256
      format:
        pattern: "^[a-zA-Z0-9._/-]+$"
  - name: "port_number"
    description: "Server port"
    required: false
    default: "8080"
    value_spec:
      type: number
      range:
        minimum: 1
        maximum: 65535
```

**デフォルト挙動:** templatizer が未定義のテンプレート変数を自動追加する際、`type: string`, `max_bytes: 256` の `value_spec` がデフォルトとして付与されます。`scaffold.yaml` に既に `value_spec` が記載されているパラメータは上書きされません。

#### 予約済みパラメータ名

templatizer は以下のパラメータ名を特別に処理します：

| パラメータ名 | 用途 |
|---|---|
| `module_path` | Go モジュールパスの変換（`go.mod` の `module` 行と `import` パスを書き換え） |
| `program_name` | プログラム名の変換（`cmd/<name>` ディレクトリのリネーム） |

上記以外のパラメータは、`.hints` ファイル内のプレースホルダ `{{param_name}}` として展開されます。

---

## Segment 2: テンプレート実体 (`catalog/templates/`)

### ファイル形式

テンプレートは **ZIP アーカイブ** として格納されます。ZIP ファイルは `templatizer` ツールにより `originals/` から自動生成されます。

```
catalog/templates/
├── axsh/
│   ├── go-standard-feature.zip
│   ├── go-standard-project.zip
│   └── go-kotoshiro-mcp-feature.zip
└── root/
    └── project-default.zip
```

### ZIP 内のファイル構造

ZIP は `originals/<project>/` ディレクトリ全体をアーカイブしたものです。`base/`（ベースファイル群）、`locale.<lang>/`（ロケールオーバーレイ、任意）、`scaffold.yaml` が含まれます。

```
# ZIP内の構造例（root/project-default の場合）
base/
  AGENTS.md
  features/
    README.md
  prompts/...
  scripts/.gitkeep
  shared/...
  work/README.md
locale.ja/
  features/
    README.md
  prompts/phases/
    README.md
  shared/...
  work/README.md
scaffold.yaml
```

> **Note**: ZIP 内の `scaffold.yaml` は scaffold 定義のコピーであり、クライアントはシャーディング YAML から情報を取得済みのため展開対象から除外する。

### テンプレート変数（`.tmpl` ファイル）

- 拡張子 `.tmpl` のファイルはテンプレートとして処理される
- 処理後、`.tmpl` 拡張子は除去される（`go.mod.tmpl` → `go.mod`）
- 利用可能な変数は `scaffold.yaml` の `template_params` で定義されたもの

```
# 例: go.mod.tmpl
module {{module_path}}

go 1.24.0
```

### ロケールオーバーレイ

`locale.<lang>/` ディレクトリに、`base/` と同じ相対パスで差分ファイルを配置します。クライアントは以下の手順でロケールを適用します。

1. ユーザーのロケールを検出（優先順: `--lang` フラグ > `LC_ALL` > `LANG`）
2. ZIP 内に該当する `locale.<lang>/` が存在するか確認
3. **存在する場合**: `base/` をコピー → `locale.<lang>/` で上書き（同名ファイルのみ置換）
4. **存在しない場合**: `base/` のみをコピー

```
# 解決順序の例（locale=ja の場合）:
# locale.ja/features/README.md が存在する → locale.ja 版を使用
# locale.ja/scripts/.gitkeep が存在しない → base/ 版を使用
```

---

## Segment 3: 配置定義 (`catalog/placements/`)

### スキーマ

```yaml
version: "1.0.0"

# 配置先ベースディレクトリ（リポジトリルートからの相対パス）
# テンプレート変数を含めることが可能: "features/{{feature_name}}"
base_dir: "."

# コンフリクト解決ポリシー
conflict_policy: "skip"    # skip | overwrite | append | error

# テンプレート処理設定
template_config:
  template_extension: ".tmpl"   # テンプレートファイルの拡張子
  strip_extension: true         # 処理後に拡張子を除去するか

# ファイル名マッピング（任意）
file_mappings:
  - source: "dot-gitignore"     # テンプレート内のファイル名
    target: ".gitignore"        # 実際のファイル名

# 後処理アクション
post_actions:
  gitignore_entries:            # .gitignore に追記するエントリ
    - "work/*"
  file_permissions:             # ファイルパーミッション設定
    - pattern: "scripts/**/*.sh"
      executable: true
```

### コンフリクト解決ポリシー

| ポリシー | 動作 | 主な用途 |
|---|---|---|
| `skip` | 既存ファイルはスキップ、新規のみ作成 | デフォルト構成（冪等実行可能） |
| `overwrite` | 既存ファイルを上書き | テンプレート更新の強制適用 |
| `append` | 既存ファイルの末尾に追記 | 設定ファイルへの追記 |
| `error` | 既存ファイルがあればエラーで中止 | 新規作成専用テンプレート |

### ファイル名マッピング (`file_mappings`)

- `.gitignore` のようなドットファイルをテンプレート内で `dot-gitignore` として管理し、配置時に `.gitignore` にリネーム
- 通常は不要（ファイル名がそのまま使える場合は省略可）

### 後処理アクション (`post_actions`)

- `gitignore_entries`: `.gitignore` に追記するパターン。既存エントリと重複する場合はスキップ
- `file_permissions`: ファイルパーミッションの設定ルール（配列）
  - `pattern`: グロブパターン（`**` による再帰マッチング対応）
  - `executable`: `true` に設定すると `0755` パーミッションを付与（`mode` の糖衣構文）
  - `mode`: 8進数文字列（例: `"0600"`, `"0644"`, `"0755"`）で明示的にパーミッションを設定
  - `executable` と `mode` の両方が指定された場合、`mode` が優先される
  - どちらか一方は必ず指定する必要がある

```yaml
post_actions:
  gitignore_entries:
    - "work/*"
  file_permissions:
    - pattern: "scripts/**/*.sh"   # スクリプトファイルに実行権限を付与
      executable: true
    - pattern: "secrets/**/*"      # シークレットファイルのアクセスを制限
      mode: "0600"
```

> **注意**: Windows 環境では `os.Chmod` による Unix パーミッションビットのサポートが限定的です。Git の `core.fileMode` 設定に依存します。

---

## templatizer ツール

`features/templatizer/` に実装されている、`originals/` から `templates/` （ZIP）を生成するためのビルドツールです。

### 実行方法

```bash
# ビルド
cd features/templatizer && go build -o ../../bin/templatizer .

# 実行
./bin/templatizer catalog.yaml
```

### 処理フロー

```
templatizer <catalog.yaml>
     │
     ├─ catalog.yaml を読み込み
     ├─ scaffolds エントリごとに処理:
     │   ├─ originals/ (original_ref) からテンポラリディレクトリにコピー
     │   ├─ template_params が定義されていれば変換パイプラインを実行:
     │   │   ├─ Step 1: クリーンアップ（不要ファイル削除）
     │   │   ├─ Step 2: Go AST 変換（モジュールパス・import書き換え＋.tmpl付与）
     │   │   ├─ Step 3: ディレクトリリネーム（cmd/<old> → cmd/<new>）
     │   │   └─ Step 4: Hints ファイル処理（置換ルール適用＋.tmpl付与）
     │   ├─ テンポラリディレクトリから ZIP を生成 → templates/ (template_ref.zip)
     │   └─ テンポラリディレクトリを削除
     └─ 完了
```

### 内部パッケージ構成

| パッケージ | 役割 |
|---|---|
| `catalog` | `catalog.yaml` の読み込みと解析 |
| `copier` | ディレクトリの再帰コピー |
| `converter` | テンプレート変換パイプライン（4ステップ） |
| `archiver` | ZIP アーカイブ生成 |

### 変換パイプライン詳細

#### Step 1: クリーンアップ (`Clean`)

以下のファイル・ディレクトリを削除します：

| 対象 | 理由 |
|---|---|
| `.git` | Git 履歴はテンプレートに不要 |
| `go.sum` | テンプレート適用時に再生成される |
| `vendor` | テンプレートに含めない |
| `bin` | ビルド生成物 |
| `.DS_Store` | macOS のメタデータ |

#### Step 2: Go AST 変換 (`TransformGoFiles`)

Go ソースコードの AST（抽象構文木）を解析し、モジュールパスの変換を行います。

- **`go.mod`**: `module <old_path>` → `module <new_path>` に書き換え
- **`*.go`**: `import` 文のパスを `old_module` → `new_module` に書き換え（前方一致）
- 変換されたファイルには `.tmpl` 拡張子を付与

#### Step 3: ディレクトリリネーム (`RenameDirectories`)

- `cmd/<old_program_name>` → `cmd/<new_program_name>` にリネーム
- ディレクトリが存在しない場合はスキップ

#### Step 4: Hints ファイル処理 (`ProcessHints`)

`.hints` ファイルは YAML 形式の置換ルールを定義し、対応するファイルに適用されます。

```yaml
# 例: Makefile.hints
replacements:
  - match: "original-value"
    replace_with: "{{module_path}}"
  - match: "old-program"
    replace_with: "{{program_name}}"
```

**処理ルール:**
1. `*.hints` ファイルを検索
2. 対応するターゲットファイル（`.hints` を除いたファイル名）を読み込み
3. `replace_with` 内の `{{param}}` プレースホルダを展開
4. `match` → 展開済み `replace_with` で文字列置換（長い match が優先）
5. ターゲットファイルに `.tmpl` 拡張子を付与
6. `.hints` ファイルを削除

---

## テンプレート作成の手順

1. **`catalog/originals/<org>/<name>/base/` にプロジェクトコードを配置**
   - ビルド・実行可能な実際のプロジェクトコードをそのまま置く
   - 空ディレクトリは `.gitkeep` で表現

2. **（任意）`.hints` ファイルを配置**
   - Go 以外のファイル（`Makefile`, `Dockerfile` 等）をテンプレート化する場合
   - 対象ファイルと同じディレクトリに `<filename>.hints` を作成

3. **`catalog/placements/<org>/<name>.yaml` に配置定義を作成**
   - `base_dir`、`conflict_policy`、`post_actions` を適切に設定

4. **`catalog.yaml` にエントリを追加**
   - `name`, `category`, `template_ref`, `original_ref`, `placement_ref` を記述
   - 必要に応じて `requirements` と `template_params` を定義

5. **`templatizer` を実行して ZIP を生成**
   - `./bin/templatizer catalog.yaml`

6. **（任意）ロケールオーバーレイを追加**
   - `catalog/originals/<org>/<name>/locale.<lang>/` を作り、言語固有ファイルのみ配置

---

## devctl scaffold の実行フロー

```
devctl scaffold [category] [name]
     │
     ├─ catalog.yaml をダウンロード
     ├─ パターン解決（name/category で検索）
     ├─ 前提条件チェック（requirements）
     ├─ placement.yaml をダウンロード
     ├─ templates/<ref>.zip をダウンロード・展開
     ├─ locale オーバーレイを適用（該当言語があれば）
     ├─ テンプレート変数を適用（.tmpl ファイル）
     ├─ 実行計画を表示（dry-run + コンフリクト判定 + パーミッション変更予定）
     ├─ ユーザー確認 [y/N]
     ├─ チェックポイント作成（git stash）
     ├─ ファイル配置（conflict_policy に従う）
     ├─ 後処理（gitignore 追記 + ファイルパーミッション適用）
     └─ 完了
```

---

## ビルド・テスト

### ビルドスクリプト

```bash
# 全体ビルド＆ユニットテスト
./scripts/process/build.sh

# 統合テスト
./scripts/process/integration_test.sh
```

`build.sh` は以下を実行します：
1. `features/*/` 配下の Go プロジェクトをビルド＆テスト → `bin/` に出力
2. `catalog/originals/` 配下の Go プロジェクトをビルド＆テスト → `bin/` に出力
