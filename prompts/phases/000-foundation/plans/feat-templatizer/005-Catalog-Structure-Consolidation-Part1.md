# 005-Catalog-Structure-Consolidation-Part1

> **Source Specification**: [005-Catalog-Structure-Consolidation.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/005-Catalog-Structure-Consolidation.md)

## Goal Description

カタログ構造の統合 Part1。`scaffold.yaml`（placement 内包）のパースとスキャン機能、`CatalogIndex` / `MetaCatalog` 型の定義、scaffold.yaml ファイルの作成を行う。

## User Review Required

> [!IMPORTANT]
> **入力の二元化期間**: Part1 では `originals/*/scaffold.yaml` のパースとスキャン機能を実装するが、`main.go` のリファクタリングは Part2 で行う。Part1 完了時点では従来の `catalog.yaml` ベースの入力も残る。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R2: scaffold 定義の originals 配置 | Proposed Changes > scaffold.yaml ファイル群 |
| R3: `placement_ref` 廃止と配置ルール内包 | Proposed Changes > catalog.go (Placement, ScaffoldDefinition 型) |
| R6: templatizer 入力変更（スキャン機能） | Proposed Changes > catalog.go (ScanScaffoldDefinitions) |
| R7: CatalogIndex / MetaCatalog 型 | Proposed Changes > catalog.go (CatalogIndex, MetaCatalog 型) |
| R1, R4, R5: templates 廃止、template_ref 変更、placements 廃止 | → Part2 で対応 |
| R7: meta.yaml / catalog.yaml 書き出し | → Part2 で対応 |
| O1: マイグレーション | → Part2 で対応（手動作成で代替） |

## Proposed Changes

### catalog パッケージ

#### [MODIFY] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)

*   **Description**: `ScaffoldDefinition`（placement 内包）のパーステスト、`ScanScaffoldDefinitions` のテスト、`CatalogIndex` 生成テストを追加
*   **Technical Design**:
    ```go
    func TestParseScaffoldDefinition(t *testing.T)
    func TestScanScaffoldDefinitions(t *testing.T)
    func TestBuildCatalogIndex(t *testing.T)
    ```
*   **Logic**:
    *   **TestParseScaffoldDefinition**: テーブル駆動テスト
        | ケース | テスト内容 |
        |---|---|
        | placement 内包 | `scaffold.yaml` に `placement` セクションを含むパース |
        | placement なし | `placement` セクション省略時のパース |
        | depends_on + template_params | 全フィールドを含むフルパース |
    *   **TestScanScaffoldDefinitions**: 
        - テスト用の一時ディレクトリに `originals/test-org/test-scaffold/scaffold.yaml` を作成
        - `ScanScaffoldDefinitions` で検出できることを確認
        - `scaffold.yaml` が存在しないディレクトリはスキップされることを確認
    *   **TestBuildCatalogIndex**:
        - Scaffold スライスから `CatalogIndex` を生成
        - category → name → shard path のマッピングが正しいことを確認

#### [MODIFY] [catalog.go](file://features/templatizer/internal/catalog/catalog.go)

*   **Description**: `Placement`、`ScaffoldDefinition`、`CatalogIndex`、`MetaCatalog` 型の追加。`ScanScaffoldDefinitions` と `BuildCatalogIndex` 関数の追加
*   **Technical Design**:
    ```go
    // Placement represents the placement rules for a scaffold.
    type Placement struct {
        BaseDir        string          `yaml:"base_dir"`
        ConflictPolicy string          `yaml:"conflict_policy"`
        TemplateConfig *TemplateConfig `yaml:"template_config,omitempty"`
        FileMappings   []interface{}   `yaml:"file_mappings,omitempty"`
        PostActions    *PostActions    `yaml:"post_actions,omitempty"`
    }

    // TemplateConfig represents template processing configuration.
    type TemplateConfig struct {
        TemplateExtension string `yaml:"template_extension"`
        StripExtension    bool   `yaml:"strip_extension"`
    }

    // PostActions represents post-processing actions after scaffold application.
    type PostActions struct {
        GitignoreEntries []string         `yaml:"gitignore_entries,omitempty"`
        FilePermissions  []FilePermission `yaml:"file_permissions,omitempty"`
    }

    // FilePermission represents a file permission rule.
    type FilePermission struct {
        Pattern    string `yaml:"pattern"`
        Executable bool   `yaml:"executable"`
    }

    // ScaffoldDefinition represents a single scaffold.yaml input file.
    // This is the developer-facing format placed in originals/.
    type ScaffoldDefinition struct {
        Name           string          `yaml:"name"`
        Category       string          `yaml:"category"`
        Description    string          `yaml:"description"`
        DependsOn      []DependencyRef `yaml:"depends_on,omitempty"`
        OriginalRef    string          `yaml:"original_ref"`
        Placement      *Placement      `yaml:"placement,omitempty"`
        TemplateParams []TemplateParam `yaml:"template_params,omitempty"`
    }

    // CatalogIndex represents the index catalog.yaml file.
    // Format: scaffolds -> category -> name -> shard path
    type CatalogIndex struct {
        Scaffolds map[string]map[string]string `yaml:"scaffolds"`
    }

    // MetaCatalog represents the meta.yaml file (renamed from MinimalCatalog).
    type MetaCatalog struct {
        Version         string `yaml:"version"`
        DefaultScaffold string `yaml:"default_scaffold"`
        UpdatedAt       string `yaml:"updated_at"`
    }

    // ParseScaffoldDefinition parses a single scaffold.yaml file.
    func ParseScaffoldDefinition(data []byte) (*ScaffoldDefinition, error) {
        var def ScaffoldDefinition
        if err := yaml.Unmarshal(data, &def); err != nil {
            return nil, fmt.Errorf("failed to parse scaffold definition: %w", err)
        }
        return &def, nil
    }

    // ScanScaffoldDefinitions walks the originals directory and loads
    // all scaffold.yaml files, returning them as Scaffold slice.
    func ScanScaffoldDefinitions(originalsDir string) ([]ScaffoldDefinition, error)
    // → filepath.WalkDir で originals/ 配下を走査
    //   "scaffold.yaml" を見つけたら ParseScaffoldDefinition で読み込み
    //   結果をスライスに追加して返す

    // BuildCatalogIndex builds a CatalogIndex from scaffolds.
    func BuildCatalogIndex(scaffolds []Scaffold) *CatalogIndex
    // → 各 scaffold の ScaffoldHash を算出
    //   ScaffoldShardPath でパスを取得
    //   category → name → path のマッピングを構築
    ```
*   **Logic (ScanScaffoldDefinitions)**:
    1. `filepath.WalkDir(originalsDir, ...)` で再帰走査
    2. ファイル名が `"scaffold.yaml"` のエントリを検出
    3. `os.ReadFile` で読み込み → `ParseScaffoldDefinition` でパース
    4. 結果を `[]ScaffoldDefinition` に追加
    5. エラー時はラップして返す
*   **Logic (BuildCatalogIndex)**:
    1. `CatalogIndex{Scaffolds: make(map[string]map[string]string)}` を初期化
    2. 各 scaffold について:
       - `ScaffoldHash(s.Category, s.Name)` でハッシュ算出
       - `ScaffoldShardPath(hash)` でパス取得
       - `index.Scaffolds[s.Category]` が nil なら `make(map[string]string)` で初期化
       - `index.Scaffolds[s.Category][s.Name] = path` で登録
    3. `*CatalogIndex` を返す

### scaffold.yaml ファイル群

#### [NEW] [scaffold.yaml](file://catalog/originals/root/project-default/scaffold.yaml)

```yaml
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

#### [NEW] [scaffold.yaml](file://catalog/originals/axsh/go-standard-project/scaffold.yaml)

```yaml
name: "axsh-go-standard"
category: "project"
description: "AXSH Go Standard Project"
depends_on:
  - category: "root"
    name: "default"
original_ref: "catalog/originals/axsh/go-standard-project"
placement:
  base_dir: "."
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions:
    file_permissions:
      - pattern: "scripts/**/*.sh"
        executable: true
```

#### [NEW] [scaffold.yaml](file://catalog/originals/axsh/go-standard-feature/scaffold.yaml)

```yaml
name: "axsh-go-standard"
category: "feature"
description: "AXSH Go Standard Feature"
depends_on:
  - category: "project"
    name: "axsh-go-standard"
original_ref: "catalog/originals/axsh/go-standard-feature"
placement:
  base_dir: "features/{{feature_name}}"
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions: {}
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myprog"
  - name: "program_name"
    description: "Program name"
    required: true
    default: "myprog"
```

#### [NEW] [scaffold.yaml](file://catalog/originals/axsh/go-kotoshiro-mcp-feature/scaffold.yaml)

```yaml
name: "axsh-go-kotoshiro-mcp"
category: "feature"
description: "AXSH Go Kotoshiro MCP Feature (Kuniumi-based)"
depends_on:
  - category: "project"
    name: "axsh-go-standard"
original_ref: "catalog/originals/axsh/go-kotoshiro-mcp-feature"
placement:
  base_dir: "features/{{feature_name}}"
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions: {}
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

## Step-by-Step Implementation Guide

- [x] **Step 1: Placement 型と ScaffoldDefinition 型の追加（TDD）**
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestParseScaffoldDefinition` テスト関数を追加（3ケース: placement 内包、placement なし、全フィールド）
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `Placement`, `TemplateConfig`, `PostActions`, `FilePermission` 構造体を追加
      - `ScaffoldDefinition` 構造体を追加
      - `ParseScaffoldDefinition` 関数を追加
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 2: ScanScaffoldDefinitions の実装（TDD）**
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestScanScaffoldDefinitions` テスト関数を追加（正常系 + scaffold.yaml なしのスキップ）
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `ScanScaffoldDefinitions` 関数を追加
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 3: CatalogIndex / MetaCatalog 型と BuildCatalogIndex の実装（TDD）**
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestBuildCatalogIndex` テスト関数を追加
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `CatalogIndex` 構造体を追加
      - `MetaCatalog` 構造体を追加（既存 `MinimalCatalog` をリネーム）
      - `BuildCatalogIndex` 関数を追加
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 4: scaffold.yaml ファイルの作成**
    - `catalog/originals/root/project-default/scaffold.yaml` を作成
    - `catalog/originals/axsh/go-standard-project/scaffold.yaml` を作成
    - `catalog/originals/axsh/go-standard-feature/scaffold.yaml` を作成
    - `catalog/originals/axsh/go-kotoshiro-mcp-feature/scaffold.yaml` を作成

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        - `TestParseScaffoldDefinition` がパスすること
        - `TestScanScaffoldDefinitions` がパスすること
        - `TestBuildCatalogIndex` がパスすること
        - 既存テストがすべてパスすること（リグレッションなし）

## Documentation

- ドキュメント更新は Part2 で一括して行う

## 継続計画について

Part2 では以下を実装する:
- R1: `templates/` 廃止と ZIP のシャーディングディレクトリ配置
- R4: `template_ref` の変更
- R5: `placements/` ディレクトリの廃止
- R7: `meta.yaml` と `catalog.yaml`（インデックス）の書き出し
- `main.go` のリファクタリング（入力を `ScanScaffoldDefinitions` に変更、出力先変更）
- ドキュメント更新
