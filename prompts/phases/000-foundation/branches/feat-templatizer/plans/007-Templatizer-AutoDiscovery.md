# 007-Templatizer-AutoDiscovery

> **Source Specification**: [007-Templatizer-AutoDiscovery.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/007-Templatizer-AutoDiscovery.md)

## Goal Description

templatizer の CLI 引数を変更し、固定パス (`baseDir + "catalog/originals"`) による originals ディレクトリの参照を廃止する。代わりに、指定ディレクトリ以下を再帰的に走査して `originals` ディレクトリを自動検出する方式に移行する。また `--help` / `-h` オプションを追加する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. 引数を「探索対象ディレクトリ」に変更 | Proposed Changes > main.go |
| 2. originals の自動探索（複数はエラー） | Proposed Changes > discovery.go |
| 3. originals 配下の scaffold.yaml 探索 | Proposed Changes > discovery.go（既存 `ScanScaffoldDefinitions` を活用） |
| 4. `--help` オプション追加 | Proposed Changes > main.go |
| 5. catalog.yaml の入力引数としての廃止 | Proposed Changes > main.go（`LoadCatalog` 呼び出し削除は不要: 既に使っていない） |
| 6. 出力先の決定ロジック | Proposed Changes > discovery.go（`DiscoveryResult.BaseDir`） |
| 7. 後方互換性 | 出力フォーマットは変更なし（既存処理パイプラインは `baseDir` の取得元が変わるのみ） |

## Proposed Changes

### catalog パッケージ (`internal/catalog`)

#### [NEW] [discovery.go](file://features/templatizer/internal/catalog/discovery.go)

*   **Description**: originals ディレクトリの自動検出ロジック
*   **Technical Design**:
    ```go
    // DiscoveryResult represents a discovered originals directory and its scaffolds.
    type DiscoveryResult struct {
        OriginalsDir string               // absolute path to the originals/ directory
        BaseDir      string               // parent of originals/ (output target)
        Definitions  []ScaffoldDefinition  // scaffolds found in this originals/
    }

    // DiscoverOriginals walks searchRoot recursively to find directories named "originals".
    // Returns error if 0 or 2+ originals directories are found.
    func DiscoverOriginals(searchRoot string) (*DiscoveryResult, error)
    ```
*   **Logic**:
    1. `filepath.WalkDir(searchRoot, ...)` で再帰走査
    2. `d.IsDir() && d.Name() == "originals"` のエントリを `found` スライスに追加
    3. `originals` ディレクトリ自体の配下はスキップする (`return fs.SkipDir`)
       - これにより `originals/xxx/originals/` のようなネスト誤検知を防ぐ
    4. 走査完了後、`len(found)` で分岐:
       - `0`: `fmt.Errorf("no originals directory found under %s", searchRoot)` を返す
       - `2+`: 全パスを改行区切りで列挙したエラーメッセージを返す
         ```
         "multiple originals directories found under %s:\n  - %s\n  - %s"
         ```
       - `1`: 正常処理を続行
    5. `baseDir = filepath.Dir(found[0])` で出力先を決定
    6. `ScanScaffoldDefinitions(found[0])` で scaffold.yaml を収集
    7. `*DiscoveryResult` を返す

#### [NEW] [discovery_test.go](file://features/templatizer/internal/catalog/discovery_test.go)

*   **Description**: `DiscoverOriginals` のテーブル駆動テスト
*   **Technical Design**:
    ```go
    func TestDiscoverOriginals(t *testing.T)
    ```
*   **Test Cases**:

    | Case | Setup | Expected |
    |---|---|---|
    | `single originals found` | `tmpDir/catalog/originals/org/test/scaffold.yaml` を作成 | `DiscoveryResult` が返る。`OriginalsDir` は `tmpDir/catalog/originals`、`BaseDir` は `tmpDir/catalog`、`Definitions` に1件 |
    | `no originals found` | `tmpDir/` に originals なし | エラー、`"no originals directory found"` を含む |
    | `multiple originals found` | `tmpDir/a/originals/` と `tmpDir/b/originals/` を作成 | エラー、`"multiple originals directories found"` を含む。両パスが列挙される |
    | `nested originals skipped` | `tmpDir/catalog/originals/` のみ作成。配下に `originals` という名のサブディレクトリは作らない（WalkDir がスキップするため実質テスト不要だが、`originals/sub/originals/` を作成してもカウントが1であることを確認） | `DiscoveryResult` が返る。検出は1件のみ |

---

### main.go

#### [MODIFY] [main.go](file://features/templatizer/main.go)

*   **Description**: 引数解析の変更、`--help` 追加、`DiscoverOriginals` の呼び出し
*   **Technical Design**:
    ```go
    // printUsage prints the help message to stderr.
    func printUsage()

    func main() {
        // 1. Parse arguments
        // 2. Call DiscoverOriginals
        // 3. Continue with existing pipeline using result.BaseDir and result.Definitions
    }
    ```
*   **Logic**:

    **引数解析 (main 関数冒頭)**:
    1. `os.Args[1:]` を走査
    2. `--help` または `-h` が含まれている場合: `printUsage()` を呼んで `os.Exit(0)`
    3. 引数が0個の場合: `printUsage()` を呼んで `os.Exit(1)`
    4. 最初の非フラグ引数を `searchRoot` とする

    **printUsage 関数**:
    ```
    templatizer - Scan originals and generate scaffold templates

    Usage:
      templatizer <search-root-dir>
      templatizer --help

    Arguments:
      <search-root-dir>  Root directory to search for 'originals' directories

    Examples:
      templatizer .
      templatizer ./catalog
    ```

    **DiscoverOriginals 呼び出し**:
    - 以下の既存コードを置換:
      ```go
      // Before:
      baseDir := os.Args[1]
      originalsDir := filepath.Join(baseDir, "catalog", "originals")
      defs, err := catalog.ScanScaffoldDefinitions(originalsDir)

      // After:
      result, err := catalog.DiscoverOriginals(searchRoot)
      // result.BaseDir を baseDir として以降使用
      // result.Definitions を defs として以降使用
      ```
    - 発見した originals ディレクトリのパスをログ出力:
      ```go
      fmt.Printf("Discovered originals: %s\n", result.OriginalsDir)
      ```
    - 以降の処理（`convertDefinitionsToScaffolds`, `processScaffold`, `generateShardFiles` 等）は `result.BaseDir` を `baseDir` として使用。変更は引数の受け渡しのみで、処理ロジック自体は変更なし。

## Step-by-Step Implementation Guide

1. **[x] `DiscoverOriginals` のテストを作成**:
   - Create `features/templatizer/internal/catalog/discovery_test.go`
   - テーブル駆動テストで4ケース（single, none, multiple, nested skip）を記述
   - `t.TempDir()` でテスト用ディレクトリを構築

2. **[x] `DiscoverOriginals` 関数を実装**:
   - Create `features/templatizer/internal/catalog/discovery.go`
   - `DiscoveryResult` 構造体と `DiscoverOriginals` 関数を実装
   - `filepath.WalkDir` で再帰走査、`originals` ディレクトリ検出
   - 0件/2件以上のエラーハンドリング
   - 1件の場合は `ScanScaffoldDefinitions` を呼び出して結果を返す

3. **[x] テスト実行で TDD サイクル完了を確認**:
   - `./scripts/process/build.sh` を実行
   - `discovery_test.go` の全ケースがパスすることを確認

4. **[x] `main.go` を変更**:
   - `printUsage()` 関数を追加
   - `main()` 関数冒頭の引数解析を変更（`--help` / `-h` 対応, `searchRoot` の取得）
   - `catalog.DiscoverOriginals(searchRoot)` の呼び出しに置換
   - 以降の処理で `result.BaseDir` を `baseDir` として使用
   - 発見時のログ出力を追加

5. **[x] ビルドと全テスト実行**:
   - `./scripts/process/build.sh` を実行
   - 全既存テストがパスすることを確認（リグレッションなし）

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```
   - 新規テスト `TestDiscoverOriginals` の全ケース（single, none, multiple, nested skip）がパス
   - 既存テスト `TestScanScaffoldDefinitions`, `TestParseCatalog` 等がリグレッションなし

## Documentation

本計画による変更は小規模であり、既存のリファレンスマニュアル等のドキュメント更新は不要です。仕様書 `007-Templatizer-AutoDiscovery.md` が本変更の根拠資料として機能します。
