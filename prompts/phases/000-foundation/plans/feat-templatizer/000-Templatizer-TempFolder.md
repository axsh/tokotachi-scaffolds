# 000-Templatizer-TempFolder

> **Source Specification**: [000-Templatizer-TempFolder.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/000-Templatizer-TempFolder.md)

## Goal Description

templatizer の ZIP 圧縮フローに「テンポラリフォルダ」を導入する。`originals/` から直接 ZIP を生成する現行方式を、一旦テンポラリフォルダへ再帰コピーしてから ZIP を生成する方式に変更する。将来的なファイル操作の拡張に備え、originals フォルダの汚染を防止する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| テンポラリフォルダの作成（`os.MkdirTemp` 使用） | Proposed Changes > main.go |
| originals からテンポラリフォルダへの再帰コピー | Proposed Changes > internal/copier/copier.go |
| テンポラリフォルダから ZIP を生成 | Proposed Changes > main.go |
| テンポラリフォルダのクリーンアップ（`defer os.RemoveAll`） | Proposed Changes > main.go |
| originals フォルダの保護（内容が変更されないこと） | Verification Plan > copier_test.go |
| テンポラリ作成・コピー・削除のログ出力 | Proposed Changes > main.go |

## Proposed Changes

### copier パッケージ（新規）

#### [NEW] [copier_test.go](file://features/templatizer/internal/copier/copier_test.go)

*   **Description**: `CopyDir` 関数のテーブル駆動テスト
*   **Technical Design**:
    ```go
    package copier

    func TestCopyDir(t *testing.T) {
        // テーブル駆動テスト: tests := []struct{ name, files map[string]string }
    }
    func TestCopyDirNotFound(t *testing.T) { ... }
    func TestCopyDirPreservesSource(t *testing.T) { ... }
    ```
*   **Logic**:
    *   **TestCopyDir**: テーブル駆動テスト。以下のケースを含む:
        | ケース名 | files (相対パス → 内容) |
        |---|---|
        | `flat files` | `{"file1.txt": "hello", "file2.txt": "world"}` |
        | `nested directories` | `{"root.txt": "root", "sub/nested.txt": "nested", "sub/deep/deep.txt": "deep"}` |
        | `empty file` | `{"empty.txt": ""}` |
        *   各ケースで: `t.TempDir()` で src を作成 → ファイルを配置 → `CopyDir(src, dest)` 呼び出し → dest 内のファイルが src と完全一致することを検証（ファイル数、パス、内容）
    *   **TestCopyDirNotFound**: 存在しないソースディレクトリを指定 → `error` が返ることを検証
    *   **TestCopyDirPreservesSource**: コピー実行後、コピー元ファイルの内容が変わっていないことを検証。 `t.TempDir()` で src を作成 → ファイル内容をメモ → `CopyDir` → src のファイル内容が変わっていないことを `assert.Equal` で確認

---

#### [NEW] [copier.go](file://features/templatizer/internal/copier/copier.go)

*   **Description**: ディレクトリの再帰コピー関数
*   **Technical Design**:
    ```go
    package copier

    // CopyDir は srcDir の内容を destDir に再帰的にコピーする。
    // destDir は存在している必要がある。
    // srcDir が存在しないかディレクトリでない場合はエラーを返す。
    func CopyDir(srcDir, destDir string) error
    ```
*   **Logic**:
    1. `os.Stat(srcDir)` で存在確認 → 存在しない場合はエラー返却
    2. `info.IsDir()` でディレクトリ確認 → ディレクトリでない場合はエラー返却
    3. `filepath.WalkDir(srcDir, ...)` で srcDir を再帰走査:
        - 各エントリについて `filepath.Rel(srcDir, path)` で相対パスを算出
        - ディレクトリの場合: `os.MkdirAll(filepath.Join(destDir, relPath), 0o755)`
        - ファイルの場合:
            1. `os.MkdirAll` で親ディレクトリを確保
            2. `os.Open(path)` でソースファイルを開く
            3. `os.Create(filepath.Join(destDir, relPath))` でコピー先ファイルを作成
            4. `io.Copy(dst, src)` でコピー
            5. 両方をClose

---

### メインプログラムの変更

#### [MODIFY] [main.go](file://features/templatizer/main.go)

*   **Description**: 各 scaffold の処理に「テンポラリフォルダ作成 → 再帰コピー → ZIP圧縮 → クリーンアップ」のフローを導入
*   **Technical Design**:
    ```go
    import (
        // 既存のインポートに追加
        "github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/copier"
    )
    ```
*   **Logic**:
    *   scaffold ループ内の処理を以下に変更:
        1. `originalDir` と `zipPath` の算出（既存のまま）
        2. `os.MkdirTemp("", "templatizer-*")` でテンポラリフォルダを作成
            - エラー時はエラーメッセージを出力して `os.Exit(1)`
        3. `defer os.RemoveAll(tempDir)` でクリーンアップを登録
            - **重要**: ループ内で defer を使うとループ終了まで発火しない問題があるため、scaffold 処理を関数に切り出すか、明示的に `os.RemoveAll` を呼ぶ
            - 推奨パターン: scaffold 処理を `processScaffold(baseDir string, s catalog.Scaffold) error` 関数として切り出し、その中で `defer` を使う
        4. `copier.CopyDir(originalDir, tempDir)` で originals をテンポラリへコピー
        5. `archiver.ZipDirectory(tempDir, zipPath)` でテンポラリから ZIP を生成
        6. ログメッセージに「テンポラリフォルダ作成 → コピー → ZIP圧縮 → クリーンアップ」の各ステップを出力

    *   **processScaffold 関数の切り出し**:
        ```go
        // processScaffold は1つの scaffold に対するテンポラリコピー→ZIP圧縮を実行する。
        // テンポラリフォルダは関数終了時にクリーンアップされる。
        func processScaffold(baseDir string, s catalog.Scaffold) error
        ```
        - この関数内で `defer os.RemoveAll(tempDir)` を使うことで、各 scaffold 処理完了後に確実にクリーンアップされる

## Step-by-Step Implementation Guide

1. [x] **copier テストの作成**:
    *   `features/templatizer/internal/copier/copier_test.go` を新規作成
    *   `TestCopyDir`（テーブル駆動: flat files, nested directories, empty file）を実装
    *   `TestCopyDirNotFound` を実装
    *   `TestCopyDirPreservesSource` を実装
    *   この時点ではコンパイルエラーになる（`CopyDir` 未実装のため）

2. [x] **copier 実装の作成**:
    *   `features/templatizer/internal/copier/copier.go` を新規作成
    *   `CopyDir(srcDir, destDir string) error` を実装
    *   `./scripts/process/build.sh` でテスト通過を確認

3. [x] **main.go の変更**:
    *   `processScaffold(baseDir string, s catalog.Scaffold) error` 関数を追加
    *   scaffold ループから `processScaffold` を呼び出すように変更
    *   `copier` パッケージの import を追加
    *   `./scripts/process/build.sh` でビルド通過を確認

4. [x] **リグレッション確認**:
    *   `./scripts/process/build.sh` で全テスト通過を確認
    *   既存の `archiver_test.go`、`catalog_test.go` がそのまま通ることを確認


## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        - `internal/copier` パッケージの全テスト（`TestCopyDir`, `TestCopyDirNotFound`, `TestCopyDirPreservesSource`）が PASS
        - 既存の `internal/archiver` と `internal/catalog` のテストが PASS（リグレッションなし）
        - ビルドが成功すること

## Documentation

本計画で影響を受ける既存ドキュメントはありません。
