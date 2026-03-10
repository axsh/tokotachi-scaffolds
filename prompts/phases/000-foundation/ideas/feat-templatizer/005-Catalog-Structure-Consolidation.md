# Catalog 構造統合: templates 廃止、meta.yaml 分離、インデックス catalog.yaml 生成

## 背景 (Background)

templatizer の出力として現在4つのディレクトリが存在する：

```
catalog/
├── originals/       ← テンプレート元ソース（開発者が編集）
├── placements/      ← 配置ルール（個別 YAML ファイル）
├── scaffolds/       ← シャーディングファイル（scaffold メタデータ）
└── templates/       ← ZIP アーカイブ（templatizer 生成物）
```

この構造には以下の課題がある：

- **`templates/` と `scaffolds/` の分散**: クライアントが scaffold を使用するには、シャーディングファイル（メタデータ）と ZIP（テンプレート）を別々の場所から取得する必要がある
- **`placements/` の冗長性**: 配置ルールは scaffold 定義と 1:1 対応しており、12〜14行程度の小さなファイル。分離する利点が薄い
- **scaffold 定義の所在が不明確**: `catalog.yaml` から消えた scaffold 定義は `scaffolds/` のハッシュパスに格納されるが、開発者が直接参照・編集するには不便
- **ハッシュ計算の障壁**: クライアントがハッシュ関数を実装できない場合、特定の scaffold に到達できない。インデックスがあればハッシュ計算不要でアクセス可能

## 要件 (Requirements)

### 必須要件

#### R1: `templates/` ディレクトリの廃止

`templates/` ディレクトリを廃止し、ZIP ファイルをシャーディングディレクトリに配置する。

**変更前:**
```
catalog/templates/root/project-default.zip
catalog/scaffolds/6/j/v/n.yaml
```

**変更後（衝突なしの場合）:**
```
catalog/scaffolds/6/j/v/
├── n.yaml                          ← メタデータ（scaffolds 配列）
└── project-default.zip              ← ZIP（元のテンプレート名を維持）
```

**変更後（ハッシュ衝突時）:**
```
catalog/scaffolds/6/j/v/
├── n.yaml                          ← メタデータ（scaffolds 配列に2エントリ）
├── project-default.zip              ← 1つ目の scaffold の ZIP
└── project-default-2.zip            ← 2つ目の scaffold の ZIP（衝突時に連番）
```

**ZIP ファイル命名ルール:**

1. ZIP のファイル名は元のテンプレート名（`original_ref` のベースネーム）を使用する
2. 同一シャーディングディレクトリ内でファイル名が衝突した場合、`{name}-{n}.zip`（n=2, 3, ...）として連番を振る
3. シャーディングファイル内の各配列要素は `template_ref` で自身の ZIP ファイルを参照する

#### R2: scaffold 定義の originals 配置

scaffold 定義 YAML を各テンプレートの `originals/` ディレクトリ直下に配置する。`base/` と同階層。

**変更前:**
```
catalog/originals/axsh/go-kotoshiro-mcp-feature/
└── base/          ← ソースコードのみ
```

**変更後:**
```
catalog/originals/axsh/go-kotoshiro-mcp-feature/
├── base/          ← ソースコード
└── scaffold.yaml  ← scaffold 定義（開発者が編集する入力ファイル）
```

- `scaffold.yaml` は templatizer の**入力**として使用される
- templatizer は `originals/` 配下の全 `scaffold.yaml` をスキャンし、`catalog.yaml`（従来の一元管理）に代わる入力源とする
- 開発者は `originals/` 配下で scaffold 定義を直接管理できる

#### R3: `placement_ref` の廃止と配置ルールの内包

`placement_ref` フィールドを廃止し、配置ルール（`placements/*.yaml` の内容）を `scaffold.yaml` に直接内包する。

**変更前（分離）:**
```yaml
# catalog/scaffolds/6/j/v/n.yaml
scaffolds:
  - name: "default"
    category: "root"
    placement_ref: "catalog/placements/default.yaml"

# catalog/placements/default.yaml（別ファイル）
base_dir: "."
conflict_policy: "skip"
template_config:
  template_extension: ".tmpl"
  strip_extension: true
post_actions:
  gitignore_entries:
    - "work/*"
```

**変更後（内包）:**
```yaml
# catalog/originals/root/project-default/scaffold.yaml
name: "default"
category: "root"
description: "Tokotachi - The First of All"
original_ref: "catalog/originals/root/project-default"
placement:
  base_dir: "."
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions:
    gitignore_entries:
      - "work/*"
```

**デメリット検討:**

| 観点 | 分離時 | 内包時 |
|---|---|---|
| ファイル数 | scaffold 定義 + placement = 2ファイル | 1ファイル |
| 編集容易性 | 2箇所を同時に編集 | 1箇所で完結 |
| 展開時の影響 | `tt` CLI で2ファイル取得が必要 | 1ファイル（シャーディング）で全情報取得 |
| placement のみの変更 | placement だけ変更可能 | scaffold.yaml を丸ごと更新（差分は小さい） |
| placement の再利用 | 複数 scaffold で共有可能 | 共有不可（ただし現状も共有されていない） |

結論: 現状 placement は scaffold と 1:1 対応であり、12〜14行程度の小さな定義。内包しても展開時に困ることはなく、管理が簡素化される。

#### R4: `template_ref` フィールドの変更

シャーディングファイル内の `template_ref` は、同ディレクトリの ZIP ファイルを指し示すように変更する。シャーディングファイルは衝突時に複数 scaffold を配列で持つため、各エントリが自身の ZIP を参照する必要があり、`template_ref` は**廃止しない**。

- **変更前**: `template_ref: "catalog/templates/root/project-default"`（別ディレクトリの ZIP を参照）
- **変更後**: `template_ref: "catalog/scaffolds/6/j/v/project-default.zip"`（同ディレクトリの ZIP を参照）

**シャーディングファイルの例（衝突時）:**
```yaml
# catalog/scaffolds/6/j/v/n.yaml
scaffolds:
  - name: "default"
    category: "root"
    template_ref: "catalog/scaffolds/6/j/v/project-default.zip"
    original_ref: "catalog/originals/root/project-default"
    # ...
  - name: "another-scaffold"
    category: "other"
    template_ref: "catalog/scaffolds/6/j/v/some-template.zip"
    original_ref: "catalog/originals/other/some-template"
    # ...
```

`original_ref` は引き続き保持する（開発者がソースコードの場所を知るために必要）。

#### R5: `placements/` ディレクトリの廃止

`placements/` ディレクトリを廃止する。配置ルールは R3 で `scaffold.yaml` に内包されるため、独立したファイルは不要になる。

#### R6: templatizer の入力変更

templatizer は従来の `catalog.yaml` ではなく、`originals/` 配下の `scaffold.yaml` ファイル群をスキャンして入力とする。

#### R7: `catalog.yaml` → `meta.yaml` リネームとインデックス `catalog.yaml` 生成

従来の最小メタデータ `catalog.yaml` を `meta.yaml` にリネームし、トップレベルに配置する。新たに `catalog.yaml` をインデックスファイルとして生成する。

**`meta.yaml`（トップレベル）:**
```yaml
version: "1.0.0"
default_scaffold: "default"
updated_at: "2026-03-10T19:00:00+09:00"
```

**`catalog.yaml`（インデックス、トップレベル）:**
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

- category をキー、その下に name をキーとして対応するシャーディングファイルパスを値とする
- ハッシュ計算を実装できないクライアントでも、このファイルを参照すれば scaffold に到達可能
- templatizer 実行時に自動生成される

**処理フロー:**

```
templatizer <catalog-dir>
     │
     ├─ catalog/ 配下の originals/*/scaffold.yaml をスキャン
     ├─ 各 scaffold.yaml を読み込み
     ├─ scaffold エントリごとに:
     │   ├─ テンプレート変換パイプライン（既存処理）
     │   └─ ZIP 生成 → catalog/scaffolds/{hash}/ に出力      ← R1
     ├─ シャーディングファイル生成:
     │   ├─ 各 scaffold のハッシュを算出
     │   ├─ catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/{h[3]}.yaml に出力
     │   └─ 既存のシャーディングディレクトリをクリーンアップ
     ├─ meta.yaml をトップレベルに生成                    ← R7
     │   └─ version, default_scaffold, updated_at
     ├─ catalog.yaml をインデックスとしてトップレベルに生成  ← R7
     │   └─ category → name → シャーディングパスのマッピング
     └─ 完了
```

### 任意要件

#### O1: `catalog.yaml` からの scaffold 定義自動分配

既存の `catalog.yaml`（フル定義形式）から `originals/*/scaffold.yaml` を自動生成するマイグレーションスクリプト/機能。一度だけ使用する移行ツール。

## 実現方針 (Implementation Approach)

### 変更対象

```
catalog/originals/*/scaffold.yaml                      # 新規: scaffold 定義（入力ファイル）
catalog/scaffolds/                                     # 変更: ZIP ファイルも配置
catalog/templates/                                     # 廃止
catalog/placements/                                    # 廃止
features/templatizer/internal/catalog/catalog.go       # 型定義の更新
features/templatizer/main.go                           # 入力スキャン処理、出力パスの変更
```

### 最終的なディレクトリ構造

```
meta.yaml                                     ← 最小メタデータ（トップレベル）
catalog.yaml                                  ← インデックス（トップレベル）
catalog/
├── originals/
│   ├── root/
│   │   └── project-default/
│   │       ├── base/                         ← ソースコード
│   │       └── scaffold.yaml                 ← scaffold 定義（入力）
│   └── axsh/
│       ├── go-standard-project/
│       │   ├── base/
│       │   └── scaffold.yaml
│       ├── go-standard-feature/
│       │   ├── base/
│       │   └── scaffold.yaml
│       └── go-kotoshiro-mcp-feature/
│           ├── base/
│           └── scaffold.yaml
└── scaffolds/                                ← シャーディング出力（配布用）
    └── 6/j/v/
        ├── n.yaml                            ← scaffold メタデータ（配列形式）
        └── project-default.zip               ← ZIP アーカイブ（元名を維持）
```

## 検証シナリオ (Verification Scenarios)

### シナリオ1: templates 廃止と ZIP 配置

1. templatizer を実行する
2. `catalog/templates/` ディレクトリが生成されないことを確認
3. `catalog/scaffolds/{hash}` ディレクトリに `.yaml` と `.zip` が同居していることを確認

### シナリオ2: scaffold.yaml の入力

1. `catalog/originals/root/project-default/scaffold.yaml` を作成する
2. templatizer を実行する
3. 該当 scaffold がシャーディングファイルに出力されることを確認

### シナリオ3: placement の内包

1. `scaffold.yaml` に `placement` セクションを含める
2. シャーディングファイルにも `placement` が含まれることを確認
3. `catalog/placements/` ディレクトリが不要であることを確認

### シナリオ4: meta.yaml と catalog.yaml の生成

1. templatizer 実行後、トップレベルに `meta.yaml` が生成されることを確認
2. `meta.yaml` に `version`, `default_scaffold`, `updated_at` のみが含まれることを確認
3. トップレベルに `catalog.yaml` がインデックスとして生成されることを確認
4. `catalog.yaml` の `scaffolds` が category → name → シャーディングパスのマッピングであることを確認
5. 全 scaffold がインデックスに含まれていることを確認

## テスト項目 (Testing for the Requirements)

### 単体テスト

| テスト対象 | テスト内容 | テストファイル |
|---|---|---|
| scaffold.yaml パース | `scaffold.yaml` 形式（placement 内包）を正しくパースできること | `internal/catalog/catalog_test.go` |
| ZIP パス導出 | シャーディングパスから ZIP パスへの変換が正しいこと | `internal/catalog/catalog_test.go` |
| インデックス生成 | `catalog.yaml` が category → name → パスのマッピングを正しく含むこと | `internal/catalog/catalog_test.go` |

### 検証コマンド

```bash
# ビルド＆ユニットテスト
./scripts/process/build.sh
```
