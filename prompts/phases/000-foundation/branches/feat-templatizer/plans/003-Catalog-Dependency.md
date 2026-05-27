# 003-Catalog-Dependency

> **Source Specification**: [003-Catalog-Dependency.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/003-Catalog-Dependency.md)

## Goal Description

`catalog.yaml` の scaffold エントリに `depends_on` フィールド（配列形式）を追加し、scaffold 間の依存関係を明示的に定義できるようにする。また、依存先の存在確認、循環依存検出、依存チェーンのトポロジカルソート解決機能を `catalog` パッケージに実装する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `depends_on` フィールドの追加（配列形式） | Proposed Changes > catalog.go (DependencyRef, Scaffold.DependsOn) |
| R1: `depends_on` の YAML パース | Proposed Changes > catalog_test.go (TestParseCatalogWithDependsOn) |
| R2: 依存チェーンの解決（トポロジカルソート） | Proposed Changes > catalog.go (ResolveDependencyChain) |
| R3: 循環依存の検出 | Proposed Changes > catalog.go (ValidateDependencies) |
| R4: 存在しない依存先の検出 | Proposed Changes > catalog.go (ValidateDependencies) |
| catalog.yaml の更新 | Proposed Changes > catalog.yaml |

## Proposed Changes

### catalog パッケージ

#### [MODIFY] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)

*   **Description**: `depends_on` フィールドのパース、バリデーション、依存チェーン解決のテストを追加
*   **Technical Design**:
    ```go
    // テスト関数一覧
    func TestParseCatalogWithDependsOn(t *testing.T)
    func TestValidateDependencies(t *testing.T)
    func TestResolveDependencyChain(t *testing.T)
    ```
*   **Logic**:
    *   **TestParseCatalogWithDependsOn**: テーブル駆動テスト
        | ケース | 入力 | 期待 |
        |---|---|---|
        | `depends_on` あり（単一依存） | scaffold に `depends_on: [{category: "root", name: "default"}]` | `DependsOn` のスライス長1、Category/Name が一致 |
        | `depends_on` あり（複数依存） | scaffold に2要素の `depends_on` 配列 | `DependsOn` のスライス長2 |
        | `depends_on` 省略 | scaffold に `depends_on` なし | `DependsOn` が nil または空スライス |
    *   **TestValidateDependencies**: テーブル駆動テスト
        | ケース | 入力 | 期待 |
        |---|---|---|
        | 正常な依存チェーン | root(依存なし) → project → feature | エラーなし |
        | 存在しない依存先 | `depends_on` に存在しない category/name | エラー（"not found" メッセージを含む） |
        | 循環依存 | A→B→A | エラー（"circular" メッセージを含む） |
        | 依存なし（全て独立） | 全 scaffold の `depends_on` が空 | エラーなし |
        | 自己参照 | A→A | エラー（"circular" メッセージを含む） |
    *   **TestResolveDependencyChain**: テーブル駆動テスト
        | ケース | 入力 | 期待順序 |
        |---|---|---|
        | 線形チェーン (feature→project→root) | feature/axsh-go-standard を起点に解決 | [root/default, project/axsh-go-standard, feature/axsh-go-standard] |
        | 依存なし | root/default を起点に解決 | [root/default] |
        | 複数依存（ダイヤモンド型） | D→B,C、B→A、C→A | [A, B, C, D]（A が最初、D が最後） |

#### [MODIFY] [catalog.go](file://features/templatizer/internal/catalog/catalog.go)

*   **Description**: `DependencyRef` 型の追加、`Scaffold` 構造体への `DependsOn` フィールド追加、バリデーション関数と依存チェーン解決関数の実装
*   **Technical Design**:
    ```go
    // DependencyRef は依存先 scaffold への参照を表す
    type DependencyRef struct {
        Category string `yaml:"category"`
        Name     string `yaml:"name"`
    }

    // Scaffold 構造体に DependsOn を追加
    type Scaffold struct {
        Name           string          `yaml:"name"`
        Category       string          `yaml:"category"`
        Description    string          `yaml:"description"`
        DependsOn      []DependencyRef `yaml:"depends_on,omitempty"`
        TemplateRef    string          `yaml:"template_ref"`
        OriginalRef    string          `yaml:"original_ref"`
        PlacementRef   string          `yaml:"placement_ref,omitempty"`
        Requirements   *Requirements   `yaml:"requirements,omitempty"`
        TemplateParams []TemplateParam `yaml:"template_params,omitempty"`
    }

    // scaffoldKey は scaffold を一意に識別するキー文字列を返す
    func scaffoldKey(category, name string) string
    // → category + "/" + name を返す

    // FindScaffold は category/name で scaffold を検索する
    func (c *Catalog) FindScaffold(category, name string) (*Scaffold, bool)
    // → scaffolds を線形探索し、category と name が一致する最初の scaffold を返す

    // ValidateDependencies は depends_on の参照整合性と循環依存を検証する
    func (c *Catalog) ValidateDependencies() error
    // → 全 scaffold の depends_on を走査:
    //   1. 各 DependencyRef の category/name が scaffolds 内に存在するか確認（なければエラー）
    //   2. DFS で循環検出（visited + inStack のツーカラーマーキング）

    // ResolveDependencyChain は指定 scaffold の依存グラフをトポロジカルソートし、
    // ルートから末端の順序で scaffold のスライスを返す
    func (c *Catalog) ResolveDependencyChain(category, name string) ([]Scaffold, error)
    // → 1. 起点の scaffold を FindScaffold で取得
    //   2. DFS で依存グラフを探索し、post-order でスタックに積む
    //   3. スタックを反転して返す（ルートが先頭、起点が末尾）
    //   4. 循環検出時はエラーを返す
    //   5. 重複削除: 同じ scaffold が複数パスから参照されても1回だけ含める
    ```
*   **Logic (ValidateDependencies)**:
    1. `scaffoldMap := map[string]*Scaffold{}` を構築（キーは `category/name`）
    2. 全 scaffold の `depends_on` を走査:
       - 各 `DependencyRef` の `category/name` が `scaffoldMap` に存在しなければ `fmt.Errorf("scaffold %q depends on %q which does not exist", ...)`
    3. DFS で循環検出:
       - `visited map[string]bool` と `inStack map[string]bool` を使用
       - ノードが `inStack` にある状態で再訪問されたら `fmt.Errorf("circular dependency detected: %s", ...)`
*   **Logic (ResolveDependencyChain)**:
    1. `FindScaffold(category, name)` で起点を取得（見つからなければエラー）
    2. DFS ヘルパー関数:
       ```
       func dfs(key string, visited, inStack map[string]bool, result *[]Scaffold):
           if inStack[key] → エラー（循環）
           if visited[key] → return（重複スキップ）
           inStack[key] = true
           for each dep in scaffold.DependsOn:
               dfs(dep.Category + "/" + dep.Name, ...)
           inStack[key] = false
           visited[key] = true
           append scaffold to result
       ```
    3. result は post-order なのでそのままルートから末端の順序になる

### catalog.yaml

#### [MODIFY] [catalog.yaml](file://catalog.yaml)

*   **Description**: 各 scaffold エントリに `depends_on` フィールドを追加
*   **Logic**:
    - `root/default`: `depends_on` なし（省略）
    - `project/axsh-go-standard`: `depends_on: [{category: "root", name: "default"}]`
    - `feature/axsh-go-standard`: `depends_on: [{category: "project", name: "axsh-go-standard"}]`
    - `feature/axsh-go-kotoshiro-mcp`: `depends_on: [{category: "project", name: "axsh-go-standard"}]`

### templatizer main

#### [MODIFY] [main.go](file://features/templatizer/main.go)

*   **Description**: catalog ロード後に `ValidateDependencies` を呼び出し、依存関係のバリデーションを実行する
*   **Logic**:
    - `catalog.LoadCatalog(catalogPath)` の後に `cat.ValidateDependencies()` を追加
    - エラー時は `fmt.Fprintf(os.Stderr, ...)` でメッセージを出力して `os.Exit(1)`

## Step-by-Step Implementation Guide

- [x] **Step 1: DependencyRef 型と Scaffold 構造体の更新**
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `DependencyRef` 構造体を `TemplateParam` の直下に追加
      - `Scaffold` 構造体に `DependsOn []DependencyRef \`yaml:"depends_on,omitempty"\`` フィールドを追加（`Description` フィールドの直後）
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestParseCatalogWithDependsOn` テスト関数を追加（3ケース）
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 2: depends_on パーステストの実行確認**
    - `TestParseCatalogWithDependsOn` がパスすることを確認
    - `TestParseCatalog` と `TestParseCatalogWithTemplateParams` が引き続きパスすることを確認（リグレッション）

- [x] **Step 3: ValidateDependencies の実装**
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestValidateDependencies` テスト関数を追加（5ケース）
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `scaffoldKey` ヘルパー関数を追加
      - `FindScaffold` メソッドを追加
      - `ValidateDependencies` メソッドを追加（存在確認 + DFS 循環検出）
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 4: ResolveDependencyChain の実装**
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestResolveDependencyChain` テスト関数を追加（3ケース）
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `ResolveDependencyChain` メソッドを追加（DFS + post-order + 重複排除）
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 5: catalog.yaml の更新**
    - Edit `catalog.yaml`:
      - `project/axsh-go-standard` に `depends_on` を追加
      - `feature/axsh-go-standard` に `depends_on` を追加
      - `feature/axsh-go-kotoshiro-mcp` に `depends_on` を追加

- [x] **Step 6: main.go にバリデーション呼び出しを追加**
    - Edit `features/templatizer/main.go`:
      - `LoadCatalog` の後に `cat.ValidateDependencies()` を呼び出す
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 7: 全体検証**
    - `./scripts/process/build.sh` で全体ビルド＆ユニットテスト
    - templatizer を直接実行して、catalog.yaml が正しくパースされ ZIP が生成されることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        - 全ての既存テストがパスすること（リグレッションなし）
        - 新規テスト `TestParseCatalogWithDependsOn`, `TestValidateDependencies`, `TestResolveDependencyChain` がパスすること

2.  **templatizer 実行確認**:
    ```bash
    ./bin/templatizer catalog.yaml
    ```
    *   **確認項目**:
        - `catalog.yaml` に `depends_on` が追加された状態でエラーなく実行が完了すること
        - 全 scaffold の ZIP が正常に生成されること

## Documentation

#### [MODIFY] [000-Reference-Manual.md](file://prompts/specifications/000-Reference-Manual.md)

*   **更新内容**:
    - Segment 1 の `catalog.yaml` スキーマに `depends_on` フィールドの定義を追加
    - 主要フィールドテーブルに `depends_on` の行を追加
    - `depends_on` の説明セクションを追加（配列形式、`category` + `name` フィールド、依存チェーン解決の概要）
