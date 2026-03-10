# catalog.yaml 依存関係 (Dependency) 定義

## 背景 (Background)

現在の `catalog.yaml` では、各 scaffold エントリは独立して定義されている。しかし実際には scaffold 間に暗黙の依存関係が存在する：

- `feature/axsh-go-standard` → `project/axsh-go-standard` を前提
- `feature/axsh-go-kotoshiro-mcp` → `project/axsh-go-standard` を前提
- `project/axsh-go-standard` → `root/default` を前提

現状、ユーザーは手動で依存順に `tt scaffold` を複数回実行する必要があり、依存関係が明示されていないため何を先に実行すべきか分かりにくい。

### 現在の課題

1. **依存関係の暗黙化**: scaffold 間の前提条件が `catalog.yaml` に記述されておらず、`requirements` フィールド（ディレクトリ・ファイル存在チェック）で間接的に推測するしかない
2. **手動での複数回実行**: feature scaffold を使うには、事前に project → root と順に scaffold を実行する必要がある
3. **ダウンロード効率**: 依存チェーンが分かれば、必要な ZIP をまとめてダウンロードしバッチ展開できるが、現状はそのメカニズムがない

## 要件 (Requirements)

### 必須要件

#### R1: `depends_on` フィールドの追加（配列形式）

`catalog.yaml` の scaffold エントリに `depends_on` フィールドを追加する。`depends_on` は `category` と `name` の2フィールドを持つオブジェクトの **配列** とし、複数の依存先を指定可能にする。

```yaml
scaffolds:
  - name: "default"
    category: "root"
    # (依存なし — ルート scaffold)

  - name: "axsh-go-standard"
    category: "project"
    depends_on:                              # ← 新フィールド（配列形式）
      - category: "root"
        name: "default"
    # ...

  - name: "axsh-go-standard"
    category: "feature"
    depends_on:
      - category: "project"
        name: "axsh-go-standard"
    # ...

  - name: "axsh-go-kotoshiro-mcp"
    category: "feature"
    depends_on:
      - category: "project"
        name: "axsh-go-standard"
    # ...
```

**`depends_on` のスキーマ:**

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `depends_on` | array of object | △ | 依存先 scaffold のリスト。省略または空配列の場合は依存なし |
| `depends_on[].category` | string | ✔ | 依存先の `category`（scaffold エントリの `category` と一致） |
| `depends_on[].name` | string | ✔ | 依存先の `name`（scaffold エントリの `name` と一致） |

#### R2: 依存チェーンの解決（トポロジカルソート）

`depends_on` を再帰的に辿り、依存グラフ全体を **トポロジカルソート** により解決する。結果は依存元（ルート）から末端の順序となる。

**例: `tt scaffold feature axsh-go-standard` の場合**

```
依存グラフ:
  feature/axsh-go-standard
    → depends_on: [{category: "project", name: "axsh-go-standard"}]
      → depends_on: [{category: "root", name: "default"}]
        → depends_on: (なし)

解決順序（トポロジカルソート結果）:
  1. root/default
  2. project/axsh-go-standard
  3. feature/axsh-go-standard
```

**複数依存の例:**

```yaml
# feature-X が project-A と project-B の両方に依存する場合
- name: "feature-x"
  category: "feature"
  depends_on:
    - category: "project"
      name: "project-a"
    - category: "project"
      name: "project-b"
```

```
解決順序:
  1. root/default           (project-a, project-b の共通依存)
  2. project/project-a      (順序は安定ソート)
  3. project/project-b
  4. feature/feature-x
```

#### R3: 循環依存の検出

依存グラフの解決時に循環を検出した場合はエラーとする。

#### R4: 存在しない依存先の検出

`depends_on` に指定された `category`/`name` の組み合わせが `scaffolds` 内に存在しない場合はエラーとする。

### 任意要件

#### O1: `tt scaffold` コマンドでのバッチ展開（参考情報）

本仕様は `catalog.yaml` のスキーマ変更と `templatizer` 側の対応が主スコープである。`tt scaffold` コマンド側でのバッチ展開（依存チェーン分の ZIP をまとめてダウンロードし、依存元から順に展開する）は、将来の `tt` CLI 側の実装で対応する。ただし、`catalog.yaml` にこの情報が記述されていることが前提条件となるため、本仕様で先にスキーマを整備する。

## 実現方針 (Implementation Approach)

### 変更対象

```
catalog.yaml                                          # depends_on フィールドの追加
features/templatizer/internal/catalog/catalog.go      # Scaffold 構造体に DependsOn 追加
features/templatizer/internal/catalog/catalog_test.go # depends_on パースのテスト追加
```

### 設計詳細

#### 1. `catalog.yaml` の更新

各 scaffold エントリに `depends_on` フィールドを配列形式で追加する。

```yaml
# 更新後の catalog.yaml
version: "1.0.0"
default_scaffold: "default"

scaffolds:
  - name: "default"
    category: "root"
    description: "Tokotachi - The First of All"
    template_ref: "catalog/templates/root/project-default"
    original_ref: "catalog/originals/root/project-default"
    placement_ref: "catalog/placements/default.yaml"
    requirements:
      directories: []
      files: []

  - name: "axsh-go-standard"
    category: "project"
    description: "AXSH Go Standard Project"
    depends_on:
      - category: "root"
        name: "default"
    template_ref: "catalog/templates/axsh/go-standard-project"
    original_ref: "catalog/originals/axsh/go-standard-project"
    placement_ref: "catalog/placements/axsh/go-standard-project.yaml"
    requirements:
      directories: ["prompts", "scripts"]
      files: []

  - name: "axsh-go-standard"
    category: "feature"
    description: "AXSH Go Standard Feature"
    depends_on:
      - category: "project"
        name: "axsh-go-standard"
    template_ref: "catalog/templates/axsh/go-standard-feature"
    original_ref: "catalog/originals/axsh/go-standard-feature"
    placement_ref: "catalog/placements/axsh/go-standard-feature.yaml"
    requirements:
      directories: ["features"]
      files: []
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        default: "github.com/axsh/tokotachi/features/myprog"
      - name: "program_name"
        description: "Program name"
        required: true
        default: "myprog"

  - name: "axsh-go-kotoshiro-mcp"
    category: "feature"
    description: "AXSH Go Kotoshiro MCP Feature (Kuniumi-based)"
    depends_on:
      - category: "project"
        name: "axsh-go-standard"
    template_ref: "catalog/templates/axsh/go-kotoshiro-mcp-feature"
    original_ref: "catalog/originals/axsh/go-kotoshiro-mcp-feature"
    placement_ref: "catalog/placements/axsh/go-kotoshiro-mcp-feature.yaml"
    requirements:
      directories: ["features"]
      files: []
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        default: "github.com/axsh/tokotachi/features/myfunction"
      - name: "program_name"
        description: "Program name"
        required: true
        default: "myfunction"
```

#### 2. Go 構造体の更新

`DependsOn` を `DependencyRef` のスライスとして定義する。

```go
// catalog.go

// DependencyRef は依存先 scaffold への参照を表す
type DependencyRef struct {
    Category string `yaml:"category"`
    Name     string `yaml:"name"`
}

type Scaffold struct {
    Name           string          `yaml:"name"`
    Category       string          `yaml:"category"`
    Description    string          `yaml:"description"`
    DependsOn      []DependencyRef `yaml:"depends_on,omitempty"`  // ← 追加（空スライス or nil なら依存なし）
    TemplateRef    string          `yaml:"template_ref"`
    OriginalRef    string          `yaml:"original_ref"`
    PlacementRef   string          `yaml:"placement_ref"`
    Requirements   Requirements    `yaml:"requirements"`
    TemplateParams []TemplateParam `yaml:"template_params,omitempty"`
}
```

#### 3. バリデーション関数の追加

`Catalog` 構造体に以下のバリデーションメソッドを追加する：

- **依存先の存在確認**: `depends_on` の各要素で参照された `category`/`name` の組み合わせが `scaffolds` 内に存在するか
- **循環依存検出**: 依存グラフを辿って循環がないか（DFS ベースの検出）

```go
// ValidateDependencies は depends_on の参照整合性と循環依存を検証する
func (c *Catalog) ValidateDependencies() error { ... }
```

### 影響範囲

- **templatizer**: `catalog.yaml` のパースに `DependsOn` フィールドが追加されるが、templatizer 自体の変換処理には影響しない（依存解決は `tt` CLI 側の責務）
- **リファレンスマニュアル**: `catalog.yaml` のスキーマ説明に `depends_on` を追記する必要がある
- **tt CLI**: 将来的にこの情報を使ってバッチ展開を実装（本仕様のスコープ外）

## 検証シナリオ (Verification Scenarios)

### シナリオ1: catalog.yaml パース確認

1. `catalog.yaml` に `depends_on` フィールドを配列形式で追加した状態で templatizer を実行する
2. エラーなく正常に ZIP が生成されることを確認する
3. `depends_on` を持たない scaffold（`root/default`）もエラーなく処理されることを確認する

### シナリオ2: 依存先の存在チェック

1. `depends_on` に存在しない `category`/`name` の組み合わせを指定する
2. `ValidateDependencies` がエラーを返すことを確認する

### シナリオ3: 循環依存の検出

1. A → B → A のような循環依存を catalog に記述する
2. `ValidateDependencies` がエラーを返すことを確認する

### シナリオ4: 依存チェーン解決（トポロジカルソート）

1. `feature/axsh-go-standard` の依存グラフを解決し、`[root/default, project/axsh-go-standard, feature/axsh-go-standard]` の順序が返ることを確認する

## テスト項目 (Testing for the Requirements)

### 単体テスト

| テスト対象 | テスト内容 | テストファイル |
|---|---|---|
| `ParseCatalog` | `depends_on` 配列が正しくパースされること | `internal/catalog/catalog_test.go` |
| `ParseCatalog` | `depends_on` 省略時に空スライスまたは nil になること | `internal/catalog/catalog_test.go` |
| `ParseCatalog` | 複数依存の `depends_on` が正しくパースされること | `internal/catalog/catalog_test.go` |
| `ValidateDependencies` | 正常な依存グラフでエラーなし | `internal/catalog/catalog_test.go` |
| `ValidateDependencies` | 存在しない依存先でエラー | `internal/catalog/catalog_test.go` |
| `ValidateDependencies` | 循環依存でエラー | `internal/catalog/catalog_test.go` |
| `ResolveDependencyChain` | 依存グラフがトポロジカルソートで解決されること | `internal/catalog/catalog_test.go` |

### 検証コマンド

```bash
# ビルド＆ユニットテスト
./scripts/process/build.sh
```
