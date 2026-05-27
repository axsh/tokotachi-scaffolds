# 005-Catalog-Structure-Consolidation-Part2

> **Source Specification**: [005-Catalog-Structure-Consolidation.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/005-Catalog-Structure-Consolidation.md)

## Goal Description

カタログ構造の統合 Part2。`main.go` のリファクタリング（入力を `ScanScaffoldDefinitions` に変更）、ZIP 出力先をシャーディングディレクトリに変更、`templates/` と `placements/` の廃止、`meta.yaml` と `catalog.yaml`（インデックス）の書き出し、ドキュメント更新を行う。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `templates/` 廃止と ZIP のシャーディング配置 | Proposed Changes > main.go (processScaffold, generateShardFiles) |
| R4: `template_ref` の変更 | Proposed Changes > main.go (generateShardFiles) |
| R5: `placements/` 廃止 | Step-by-Step > Step 3 |
| R6: templatizer 入力変更 | Proposed Changes > main.go (main 関数の入力切替) |
| R7: meta.yaml と catalog.yaml（インデックス）生成 | Proposed Changes > main.go (writeMetaCatalog, writeCatalogIndex) |
| R2, R3: scaffold 定義の originals 配置、placement 内包 | → Part1 で対応済み |

## Proposed Changes

### templatizer main

#### [MODIFY] [main.go](file://features/templatizer/main.go)

*   **Description**: 入力を `ScanScaffoldDefinitions` ベースに変更。ZIP 出力先をシャーディングディレクトリに変更。`meta.yaml` と `catalog.yaml`（インデックス）を生成
*   **Technical Design**:
    ```go
    import (
        "path/filepath"
        "time"

        "gopkg.in/yaml.v3"
    )

    func main() {
        // 1. 引数: catalog ディレクトリのパス
        catalogDir := os.Args[1]

        // 2. originals/ 配下の scaffold.yaml をスキャン
        originalsDir := filepath.Join(catalogDir, "catalog", "originals")
        defs, err := catalog.ScanScaffoldDefinitions(originalsDir)
        // → ScaffoldDefinition スライス

        // 3. ScaffoldDefinition → Scaffold に変換
        scaffolds := convertDefinitionsToScaffolds(defs)

        // 4. 依存関係バリデーション
        tempCat := &catalog.Catalog{Scaffolds: scaffolds}
        tempCat.ValidateDependencies()

        // 5. 各 scaffold を処理（ZIP生成は一時保留）
        for _, s := range scaffolds {
            processScaffold(catalogDir, s)
            // ZIP は一時的に tempDir/{basename}.zip に生成
        }

        // 6. シャーディングファイル + ZIP をシャーディングディレクトリに配置
        cleanScaffoldsDir(catalogDir)
        generateShardFiles(catalogDir, scaffolds)

        // 7. meta.yaml 生成
        writeMetaCatalog(catalogDir, "1.0.0", "default")

        // 8. catalog.yaml（インデックス）生成
        index := catalog.BuildCatalogIndex(scaffolds)
        writeCatalogIndex(catalogDir, index)
    }

    // convertDefinitionsToScaffolds converts ScaffoldDefinition slice to Scaffold slice.
    func convertDefinitionsToScaffolds(defs []catalog.ScaffoldDefinition) []catalog.Scaffold

    // writeMetaCatalog writes meta.yaml to the top-level directory.
    func writeMetaCatalog(baseDir, version, defaultScaffold string) error
    // → MetaCatalog{Version, DefaultScaffold, UpdatedAt: time.Now().Format(time.RFC3339)}
    //   filepath.Join(baseDir, "meta.yaml") に書き出し

    // writeCatalogIndex writes catalog.yaml (index) to the top-level directory.
    func writeCatalogIndex(baseDir string, index *catalog.CatalogIndex) error
    // → yaml.Marshal(index) → filepath.Join(baseDir, "catalog.yaml") に書き出し
    ```
*   **Logic (main 関数)**:
    1. `os.Args[1]` からベースディレクトリ取得（従来の `catalog.yaml` パスではなくディレクトリパス）
    2. `catalog.ScanScaffoldDefinitions(filepath.Join(baseDir, "catalog", "originals"))` でスキャン
    3. `convertDefinitionsToScaffolds` で `ScaffoldDefinition` → `Scaffold` に変換
    4. `ValidateDependencies` で依存関係バリデーション
    5. 各 scaffold を処理（テンプレート変換 + ZIP 生成）
    6. `cleanScaffoldsDir` → `generateShardFiles`（ZIP をシャーディングディレクトリに配置）
    7. `writeMetaCatalog` で `meta.yaml` を生成
    8. `writeCatalogIndex` で `catalog.yaml`（インデックス）を生成

*   **Logic (generateShardFiles の変更)**:
    - ZIP 出力先: `catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/{h[3]}/{basename}.zip`
    - `template_ref`: 上記パスを設定
    - ZIP ファイル名衝突時: `{basename}-{n}.zip`（n=2, 3, ...）として連番
    - ZIP ファイル名の決定:
      1. `original_ref` のベースネーム（最後のパス要素）を取得
      2. 同ディレクトリ内に同名ファイルがあれば `-{n}` サフィックスを付与
      3. 各 scaffold エントリの `template_ref` に確定パスを設定

*   **Logic (convertDefinitionsToScaffolds)**:
    ```go
    func convertDefinitionsToScaffolds(defs []catalog.ScaffoldDefinition) []catalog.Scaffold {
        scaffolds := make([]catalog.Scaffold, len(defs))
        for i, d := range defs {
            scaffolds[i] = catalog.Scaffold{
                Name:           d.Name,
                Category:       d.Category,
                Description:    d.Description,
                DependsOn:      d.DependsOn,
                OriginalRef:    d.OriginalRef,
                TemplateParams: d.TemplateParams,
                // TemplateRef は generateShardFiles で設定
                // PlacementRef は廃止（Placement は ScaffoldDefinition に内包）
            }
        }
        return scaffolds
    }
    ```

## Step-by-Step Implementation Guide

- [x] **Step 1: main.go のリファクタリング — 入力変更**
    - Edit `features/templatizer/main.go`:
      - `main()` の入力を `catalog.ScanScaffoldDefinitions` ベースに変更
      - `convertDefinitionsToScaffolds` 関数を追加
      - `ValidateDependencies` の呼び出しを新しいフローに合わせる
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 2: ZIP 出力先とシャーディング処理の変更**
    - Edit `features/templatizer/main.go`:
      - `processScaffold` の ZIP 出力先をシャーディングディレクトリに変更
      - `generateShardFiles` を更新（ZIP ファイル名衝突対応、`template_ref` 設定）
      - `writeMinimalCatalog` を `writeMetaCatalog` にリネーム（`meta.yaml` に出力）
      - `writeCatalogIndex` 関数を追加（`catalog.yaml` インデックス出力）
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 3: templatizer 実行と結合検証**
    - `./bin/templatizer .` を実行（ベースディレクトリ指定）
    - 検証項目:
      - `catalog/templates/` が生成されないこと
      - `catalog/scaffolds/{hash}/` に `.yaml` と `.zip` が同居すること
      - `meta.yaml` がトップレベルに生成されること
      - `catalog.yaml` がインデックスとしてトップレベルに生成されること
      - `catalog/placements/` は既存のまま（手動削除対象）

- [x] **Step 4: ドキュメント更新**
    - Edit `prompts/specifications/000-Reference-Manual.md`:
      - `meta.yaml` のスキーマを追加
      - `catalog.yaml`（インデックス）の形式を追加
      - `scaffold.yaml`（originals 配下の入力ファイル）の形式を追加
      - `templates/` と `placements/` の廃止を反映
      - `template_ref` の変更を反映

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **templatizer 実行による結合検証**:
    ```bash
    ./bin/templatizer .
    ```
    *   **確認項目**:
        - `catalog/scaffolds/` 以下に `.yaml` と `.zip` が同居
        - `meta.yaml` がトップレベルに存在（`version`, `default_scaffold`, `updated_at`）
        - `catalog.yaml` がインデックスとしてトップレベルに存在（category → name → path）
        - `catalog/templates/` が生成されない

3.  **シャーディングファイル内容確認**:
    ```bash
    find catalog/scaffolds -name "*.yaml" -exec echo "--- {} ---" \; -exec cat {} \;
    cat meta.yaml
    cat catalog.yaml
    ```
    *   `template_ref` が `catalog/scaffolds/{hash}/{basename}.zip` 形式であること
    *   全 scaffold がインデックスに含まれていること

## Documentation

#### [MODIFY] [000-Reference-Manual.md](file://prompts/specifications/000-Reference-Manual.md)

*   **更新内容**:
    - `meta.yaml` のスキーマ定義を追加
    - `catalog.yaml`（インデックス）の形式を追加
    - `scaffold.yaml`（originals 配下）の形式と placement 内包を追記
    - `templates/` と `placements/` の廃止を反映
    - `template_ref` の変更（シャーディングディレクトリ内 ZIP を参照）を反映
    - templatizer の処理フロー図を更新
