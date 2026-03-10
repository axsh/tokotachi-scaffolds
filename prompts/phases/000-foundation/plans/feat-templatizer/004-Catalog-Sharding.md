# 004-Catalog-Sharding

> **Source Specification**: [004-Catalog-Sharding.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/004-Catalog-Sharding.md)

## Goal Description

templatizer の出力フローに scaffold 定義のハッシュベースシャーディングを追加する。FNV-1a 32ビット + base-36 (4文字) のハッシュに基づき、各 scaffold 定義を `catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/{h[3]}.yaml` に分割出力する。また、templatizer 実行後の `catalog.yaml` を最小メタデータ（`version`, `default_scaffold`, `updated_at`）のみに縮小する。

## User Review Required

> [!IMPORTANT]
> **catalog.yaml の二重状態**: templatizer 実行前は scaffold 定義を含むフル形式、実行後は最小メタデータのみになります。Git 上の `catalog.yaml` は最小メタデータ版がコミットされることを想定しています。開発中に scaffold 定義を変更する場合は、手元のフル定義版 `catalog.yaml` を使い、都度 templatizer を実行してシャーディングファイルを再生成します。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: FNV-1a 32ビット + base-36 ハッシュ | Proposed Changes > catalog.go (ScaffoldHash) |
| R1: 1文字区切りディレクトリ構造 | Proposed Changes > catalog.go (ScaffoldShardPath) |
| R2: 個別 scaffold YAML の配列形式 | Proposed Changes > catalog.go (ShardFile 型) |
| R3: catalog.yaml の最小化 | Proposed Changes > catalog.go (MinimalCatalog 型), main.go |
| R4: templatizer によるシャーディング出力 | Proposed Changes > main.go (generateShardFiles, writeMinimalCatalog) |
| R4: 既存シャーディングファイルのクリーンアップ | Proposed Changes > main.go (cleanScaffoldsDir) |
| O1: ハッシュ衝突の統計表示 | Proposed Changes > main.go (generateShardFiles 内) |

## Proposed Changes

### catalog パッケージ

#### [MODIFY] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)

*   **Description**: `ScaffoldHash` と `ScaffoldShardPath` のテストを追加
*   **Technical Design**:
    ```go
    func TestScaffoldHash(t *testing.T)
    func TestScaffoldShardPath(t *testing.T)
    ```
*   **Logic**:
    *   **TestScaffoldHash**: テーブル駆動テスト
        | ケース | テスト内容 |
        |---|---|
        | 既知入力のハッシュ値 | `ScaffoldHash("root", "default")` が4文字 base-36 を返す |
        | 冪等性 | 同じ入力で2回呼び出し → 同じ結果 |
        | 4文字固定（0パディング） | ハッシュ値が小さい入力でも4文字であること |
        | 文字セット | 結果が `[0-9a-z]` のみで構成されていること |
    *   **TestScaffoldShardPath**: テーブル駆動テスト
        | ケース | 入力 | 期待 |
        |---|---|---|
        | 4文字ハッシュ "a3k9" | `ScaffoldShardPath("a3k9")` | `"catalog/scaffolds/a/3/k/9.yaml"` |

#### [MODIFY] [catalog.go](file://features/templatizer/internal/catalog/catalog.go)

*   **Description**: ハッシュ関数、シャードパス関数、シャーディング関連の型を追加
*   **Technical Design**:
    ```go
    import (
        "hash/fnv"
        "strconv"
    )

    // ScaffoldHash returns a 4-character base-36 hash string for the given
    // category and name. Uses FNV-1a 32-bit with modulo 36^4 (1,679,616).
    func ScaffoldHash(category, name string) string {
        h := fnv.New32a()
        h.Write([]byte(category + "/" + name))
        v := h.Sum32() % 1679616 // 36^4
        s := strconv.FormatUint(uint64(v), 36)
        return fmt.Sprintf("%04s", s)
    }

    // ScaffoldShardPath returns the relative file path for the shard file
    // based on the given hash. Format: catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/{h[3]}.yaml
    func ScaffoldShardPath(hash string) string {
        return fmt.Sprintf("catalog/scaffolds/%c/%c/%c/%c.yaml",
            hash[0], hash[1], hash[2], hash[3])
    }

    // ShardFile represents a single shard YAML file containing one or more scaffolds.
    type ShardFile struct {
        Scaffolds []Scaffold `yaml:"scaffolds"`
    }

    // MinimalCatalog represents the minimized catalog.yaml after sharding.
    type MinimalCatalog struct {
        Version         string `yaml:"version"`
        DefaultScaffold string `yaml:"default_scaffold"`
        UpdatedAt       string `yaml:"updated_at"`
    }

    // Catalog 構造体に DefaultScaffold フィールドを追加
    type Catalog struct {
        Version         string     `yaml:"version"`
        DefaultScaffold string     `yaml:"default_scaffold,omitempty"`
        Scaffolds       []Scaffold `yaml:"scaffolds"`
    }
    ```
*   **Logic**:
    *   `ScaffoldHash`: `category + "/" + name` を FNV-1a 32 ビットでハッシュ → `% 1679616` → `strconv.FormatUint(v, 36)` → `fmt.Sprintf("%04s", s)` で 0 パディング
    *   `ScaffoldShardPath`: 4文字ハッシュを1文字ずつ分割して3階層ディレクトリ + ファイル名を構成

### templatizer main

#### [MODIFY] [main.go](file://features/templatizer/main.go)

*   **Description**: 既存の scaffold 処理ループの後に、シャーディングファイル生成と `catalog.yaml` 最小化の処理を追加
*   **Technical Design**:
    ```go
    // generateShardFiles groups scaffolds by hash, writes shard YAML files,
    // and reports any hash collisions.
    func generateShardFiles(baseDir string, scaffolds []catalog.Scaffold) error
    // → 1. 各 scaffold の ScaffoldHash を算出
    //   2. map[string][]catalog.Scaffold にグルーピング
    //   3. 衝突がある場合は標準出力にレポート
    //   4. 各グループを ShardFile として YAML にマーシャル
    //   5. ScaffoldShardPath で算出したパスにディレクトリを作成して書き出し

    // writeMinimalCatalog writes a minimal catalog.yaml with only metadata fields.
    func writeMinimalCatalog(catalogPath string, cat *catalog.Catalog) error
    // → MinimalCatalog{Version, DefaultScaffold, UpdatedAt: 現在時刻 RFC3339} を YAML に書き出し

    // cleanScaffoldsDir removes existing catalog/scaffolds/ directory before regeneration.
    func cleanScaffoldsDir(baseDir string) error
    // → os.RemoveAll(filepath.Join(baseDir, "catalog", "scaffolds"))
    ```
*   **Logic (main 関数内の追加)**:
    1. 既存の scaffold 処理ループ完了後:
    2. `cleanScaffoldsDir(baseDir)` で既存シャーディングファイルを一掃
    3. `generateShardFiles(baseDir, cat.Scaffolds)` でシャーディングファイル生成
    4. `writeMinimalCatalog(catalogPath, cat)` で `catalog.yaml` を最小化
    5. 完了メッセージ出力

## Step-by-Step Implementation Guide

- [x] **Step 1: ScaffoldHash と ScaffoldShardPath の実装（TDD）**
    - Edit `features/templatizer/internal/catalog/catalog_test.go`:
      - `TestScaffoldHash` テスト関数を追加（4ケース: 既知ハッシュ、冪等性、4文字固定、文字セット検証）
      - `TestScaffoldShardPath` テスト関数を追加（1ケース: パス構造確認）
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - import に `"hash/fnv"` と `"strconv"` を追加
      - `ScaffoldHash` 関数を追加
      - `ScaffoldShardPath` 関数を追加
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 2: ShardFile 型と MinimalCatalog 型の追加**
    - Edit `features/templatizer/internal/catalog/catalog.go`:
      - `ShardFile` 構造体を追加
      - `MinimalCatalog` 構造体を追加
      - `Catalog` 構造体に `DefaultScaffold string \`yaml:"default_scaffold,omitempty"\`` を追加
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 3: main.go にシャーディング出力処理の追加**
    - Edit `features/templatizer/main.go`:
      - import に `"time"`, `"gopkg.in/yaml.v3"` を追加
      - `cleanScaffoldsDir` 関数を追加
      - `generateShardFiles` 関数を追加（グルーピング、衝突表示、YAML 出力）
      - `writeMinimalCatalog` 関数を追加（最小メタデータ YAML 出力）
      - `main()` の scaffold 処理ループ後に上記3関数を呼び出す
    - ビルド確認: `./scripts/process/build.sh`

- [x] **Step 4: templatizer 実行による結合検証**
    - `./bin/templatizer catalog.yaml` を実行
    - `catalog/scaffolds/` 以下にシャーディングファイルが生成されることを確認
    - `catalog.yaml` が最小メタデータのみになっていることを確認
    - 各シャーディングファイルの内容が `scaffolds` 配列形式であることを確認
    - 全 scaffold がいずれかのシャーディングファイルに含まれることを確認

- [x] **Step 5: ドキュメント更新**
    - Edit `prompts/specifications/000-Reference-Manual.md`:
      - シャーディング構造の説明セクションを追加
      - ハッシュ計算式の記載

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        - 全ての既存テストがパスすること（リグレッションなし）
        - `TestScaffoldHash` と `TestScaffoldShardPath` がパスすること

2.  **templatizer 実行による結合検証**:
    ```bash
    ./bin/templatizer catalog.yaml
    ```
    *   **確認項目**:
        - `catalog/scaffolds/` 以下にシャーディングファイルが生成される
        - `catalog.yaml` が `version`, `default_scaffold`, `updated_at` のみになる
        - 各シャーディング YAML が `scaffolds` 配列形式である
        - 4つの scaffold 全てがシャーデイングファイルに含まれる

3.  **シャーディングファイルの内容確認**:
    ```bash
    find catalog/scaffolds -name "*.yaml" -exec echo "--- {} ---" \; -exec cat {} \;
    ```
    *   scaffold の数と内容が元の `catalog.yaml` と一致することを目視確認

## Documentation

#### [MODIFY] [000-Reference-Manual.md](file://prompts/specifications/000-Reference-Manual.md)

*   **更新内容**:
    - 「シャーディング構造」セクションを追加
    - ハッシュ関数（FNV-1a 32ビット + base-36、4文字）の計算式を記載
    - ディレクトリ構造（`catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/{h[3]}.yaml`）の説明
    - 最小化された `catalog.yaml` のスキーマ（`version`, `default_scaffold`, `updated_at`）
    - シャーディングファイルのフォーマット（`scaffolds` 配列形式）
