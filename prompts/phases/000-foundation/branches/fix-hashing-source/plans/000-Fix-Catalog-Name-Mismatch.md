# 000-Fix-Catalog-Name-Mismatch

> **Source Specification**: [000-Fix-Catalog-Name-Mismatch.md](file://prompts/phases/000-foundation/ideas/fix-hashing-source/000-Fix-Catalog-Name-Mismatch.md)

## Goal Description

ルートの `catalog.yaml` の scaffold 名が scaffold.yaml の `name` フィールドと一致せず、`List()` API が存在しないシャードファイルを参照する問題を修正する。templatizer の出力先を `catalog/` から ルートに変更し、`catalog/catalog.yaml` と `catalog/meta.yaml` を廃止する。テストコード内の古い名前も修正する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. ルートの `catalog.yaml` と `meta.yaml` を templatizer で正しく生成。`catalog/catalog.yaml` と `catalog/meta.yaml` は廃止 | Proposed Changes > `main.go` |
| 2. scaffold.yaml の `name` フィールドを正の情報源とする | templatizer はすでに scaffold.yaml の `name` を使用している。出力先変更により自動的に達成 |
| 3. テストコードの修正 | Proposed Changes > `catalog_test.go` |
| 4. templatizer のメタ・インデックス出力先をルートに変更 | Proposed Changes > `main.go` |

## Proposed Changes

### templatizer テスト

#### [MODIFY] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)
*   **Description**: テストデータ内の古い名前 `axsh-go-kotoshiro-mcp` を正しい名前 `kotoshiro-go-mcp` に修正する。
*   **Technical Design**:
    *   修正箇所は3箇所:
        *   L538: `TestScaffoldHash` の "always 4 characters" テストケースの入力データ
        *   L709: `TestBuildCatalogIndex` のテストデータ
        *   L721: `TestBuildCatalogIndex` のアサーション
*   **Logic**:
    *   `{"feature", "axsh-go-kotoshiro-mcp"}` → `{"feature", "kotoshiro-go-mcp"}`
    *   `{Name: "axsh-go-kotoshiro-mcp", Category: "feature"}` → `{Name: "kotoshiro-go-mcp", Category: "feature"}`
    *   `require.Contains(t, index.Scaffolds["feature"], "axsh-go-kotoshiro-mcp")` → `require.Contains(t, index.Scaffolds["feature"], "kotoshiro-go-mcp")`

---

### templatizer 本体

#### [MODIFY] [main.go](file://features/templatizer/main.go)
*   **Description**: `writeCatalogIndex` と `writeMetaCatalog` の呼び出し先を `baseDir`（`catalog/`）から `repoRoot`（ルート）に変更する。
*   **Technical Design**:
    *   現在のコード (L110-L121):
        ```go
        // Write meta.yaml to top-level.
        if err := writeMetaCatalog(baseDir, "1.0.0", "default"); err != nil { ... }
        // Write catalog.yaml (index) to top-level.
        index := catalog.BuildCatalogIndex(scaffolds)
        if err := writeCatalogIndex(baseDir, index); err != nil { ... }
        ```
    *   変更後:
        ```go
        // Write meta.yaml to repo root.
        if err := writeMetaCatalog(repoRoot, "1.0.0", "default"); err != nil { ... }
        // Write catalog.yaml (index) to repo root.
        index := catalog.BuildCatalogIndex(scaffolds)
        if err := writeCatalogIndex(repoRoot, index); err != nil { ... }
        ```
    *   引数を `baseDir` から `repoRoot` に変更するだけ。関数シグネチャの変更は不要。
*   **Logic**:
    *   `baseDir` は `catalog/` ディレクトリを指す（L61）
    *   `repoRoot` は `filepath.Dir(baseDir)` でその親（リポジトリルート）を指す（L64）
    *   出力先ディレクトリを `baseDir` → `repoRoot` に変更することで、`catalog.yaml` と `meta.yaml` がルートに出力される
    *   `catalog/catalog.yaml` と `catalog/meta.yaml` には出力されなくなる

---

### カタログファイル廃止

#### [DELETE] [catalog.yaml](file://catalog/catalog.yaml)
*   **Description**: templatizer 生成の `catalog/catalog.yaml` を削除する。以降は templatizer がルートに生成するため不要。

#### [DELETE] [meta.yaml](file://catalog/meta.yaml)
*   **Description**: templatizer 生成の `catalog/meta.yaml` を削除する。以降は templatizer がルートに生成するため不要。

## Step-by-Step Implementation Guide

1.  **テストコードの修正**:
    *   Edit `features/templatizer/internal/catalog/catalog_test.go`:
        *   L538: `{"feature", "axsh-go-kotoshiro-mcp"}` → `{"feature", "kotoshiro-go-mcp"}`
        *   L709: `{Name: "axsh-go-kotoshiro-mcp", Category: "feature"}` → `{Name: "kotoshiro-go-mcp", Category: "feature"}`
        *   L721: `"axsh-go-kotoshiro-mcp"` → `"kotoshiro-go-mcp"`

2.  **templatizer 出力先の変更**:
    *   Edit `features/templatizer/main.go`:
        *   L111: `writeMetaCatalog(baseDir, ...)` → `writeMetaCatalog(repoRoot, ...)`
        *   L118: `writeCatalogIndex(baseDir, index)` → `writeCatalogIndex(repoRoot, index)`

3.  **廃止ファイルの削除**:
    *   Delete `catalog/catalog.yaml`
    *   Delete `catalog/meta.yaml`

4.  **ビルドと単体テストの実行**:
    *   `./scripts/process/build.sh` を実行して全テストが通ることを確認

5.  **templatizer を実行して検証**:
    *   templatizer を実行し、ルートに `catalog.yaml` と `meta.yaml` が正しく生成されることを確認
    *   `catalog/catalog.yaml` と `catalog/meta.yaml` が生成されていないことを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: 全テスト（特に `TestScaffoldHash`, `TestBuildCatalogIndex`）が PASS であること

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: 既存の統合テストがすべて PASS であること

## Documentation

本計画で影響を受ける既存ドキュメントはありません。
