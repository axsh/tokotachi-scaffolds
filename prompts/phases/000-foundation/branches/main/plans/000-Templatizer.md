# 000-Templatizer

> **Source Specification**: [000-Templatizer.md](file://prompts/phases/000-foundation/ideas/main/000-Templatizer.md)

## Goal Description

`features/templatizer` を、`catalog.yaml` を読み取り、各 scaffold の `original_ref` ディレクトリを ZIP 圧縮して `template_ref + .zip` のファイルとして出力する CLI ツールとして実装する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| catalog.yaml の解析 | Proposed Changes > internal/catalog |
| パスの取得（template_ref, original_ref） | Proposed Changes > internal/catalog |
| ZIP 圧縮（original_ref → template_ref + .zip） | Proposed Changes > internal/archiver |
| CLI インターフェース | Proposed Changes > main.go |
| 進捗ログ出力 | Proposed Changes > main.go |
| 既存ZIPの上書き | Proposed Changes > internal/archiver |
| エラー時の exit code 1 | Proposed Changes > main.go |

## Proposed Changes

### internal/catalog パッケージ

#### [NEW] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)
*   **Description**: catalog.yaml 解析のテーブル駆動テスト
*   **Technical Design**:
    ```go
    package catalog

    // TestParseCatalog tests YAML parsing of catalog.yaml
    func TestParseCatalog(t *testing.T) { ... }
    ```
*   **Logic**:
    *   テストケース:
        1.  **正常系**: 3つの scaffold エントリを含む YAML 文字列を解析し、各エントリの `Name`, `TemplateRef`, `OriginalRef` が正しく取得できること
        2.  **空の scaffolds**: `scaffolds: []` の場合に空のスライスを返すこと
        3.  **異常系**: 不正な YAML を渡した場合にエラーを返すこと
    *   テスト用の YAML は文字列リテラルとして用意し、`ParseCatalog([]byte)` に渡す

---

#### [NEW] [catalog.go](file://features/templatizer/internal/catalog/catalog.go)
*   **Description**: `catalog.yaml` を解析し、scaffold のリストを返す
*   **Technical Design**:
    ```go
    package catalog

    // Scaffold represents a single scaffold entry from catalog.yaml.
    type Scaffold struct {
        Name        string `yaml:"name"`
        Category    string `yaml:"category"`
        Description string `yaml:"description"`
        TemplateRef string `yaml:"template_ref"`
        OriginalRef string `yaml:"original_ref"`
    }

    // Catalog represents the top-level catalog.yaml structure.
    type Catalog struct {
        Version  string     `yaml:"version"`
        Scaffolds []Scaffold `yaml:"scaffolds"`
    }

    // ParseCatalog parses the YAML bytes and returns a Catalog.
    // Returns error if the YAML is invalid.
    func ParseCatalog(data []byte) (*Catalog, error) { ... }

    // LoadCatalog reads a catalog.yaml file and returns a Catalog.
    // Returns error if the file cannot be read or parsed.
    func LoadCatalog(path string) (*Catalog, error) { ... }
    ```
*   **Logic**:
    *   `ParseCatalog`: `yaml.Unmarshal` で `data` を `Catalog` 構造体にデコード。エラーがあればラップして返す
    *   `LoadCatalog`: `os.ReadFile` でファイルを読み、`ParseCatalog` に渡す

---

### internal/archiver パッケージ

#### [NEW] [archiver_test.go](file://features/templatizer/internal/archiver/archiver_test.go)
*   **Description**: ZIP 圧縮のテーブル駆動テスト
*   **Technical Design**:
    ```go
    package archiver

    // TestZipDirectory tests ZIP creation from a directory
    func TestZipDirectory(t *testing.T) { ... }

    // TestZipDirectoryNotFound tests error for non-existent directory
    func TestZipDirectoryNotFound(t *testing.T) { ... }
    ```
*   **Logic**:
    *   テストケース:
        1.  **正常系**: `t.TempDir()` にテスト用ファイル群を作成 → `ZipDirectory` で ZIP 化 → `archive/zip.OpenReader` で展開 → ファイル名と内容が元と一致することを検証
            *   サブディレクトリを含むケースも検証
            *   空ファイルも検証
        2.  **異常系**: 存在しないディレクトリパスを渡した場合に `error` が返ること

---

#### [NEW] [archiver.go](file://features/templatizer/internal/archiver/archiver.go)
*   **Description**: 指定ディレクトリを再帰的に ZIP 圧縮する
*   **Technical Design**:
    ```go
    package archiver

    // ZipDirectory creates a ZIP archive of srcDir and writes it to destPath.
    // The archive contains files with paths relative to srcDir.
    // If destPath already exists, it will be overwritten.
    // Returns error if srcDir does not exist or is not a directory.
    func ZipDirectory(srcDir, destPath string) error { ... }
    ```
*   **Logic**:
    1.  `os.Stat(srcDir)` でディレクトリ存在・種別を検証。存在しない or ディレクトリでない場合はエラー
    2.  出力先の親ディレクトリを `os.MkdirAll` で作成
    3.  `os.Create(destPath)` で出力ファイルを作成
    4.  `zip.NewWriter` を作成
    5.  `filepath.WalkDir(srcDir, ...)` で再帰走査:
        *   ディレクトリ自体はスキップ
        *   各ファイルについて:
            *   `filepath.Rel(srcDir, path)` で相対パスを取得
            *   `filepath.ToSlash` でパス区切りを `/` に統一（Windows対応）
            *   `zip.Writer.Create(relPath)` でエントリ作成
            *   `os.Open` → `io.Copy` でファイル内容をコピー
    6.  `zip.Writer.Close()` で ZIP を確定

---

### エントリポイント

#### [MODIFY] [main.go](file://features/templatizer/main.go)
*   **Description**: CLI 引数処理と、catalog 解析 → ZIP 生成のオーケストレーション
*   **Technical Design**:
    ```go
    package main

    import (
        "fmt"
        "log"
        "os"
        "path/filepath"

        "github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/archiver"
        "github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
    )

    func main() {
        // Parse CLI args: expect catalog.yaml path as first argument
        // Load catalog
        // For each scaffold, call archiver.ZipDirectory
    }
    ```
*   **Logic**:
    1.  `os.Args` から `catalog.yaml` のパスを取得。引数なしの場合はusageメッセージを表示して `exit(1)`
    2.  `catalog.LoadCatalog(catalogPath)` でカタログを読み込み
    3.  `catalog.yaml` のあるディレクトリを基準ディレクトリ (`baseDir`) として取得（`filepath.Dir(catalogPath)`）
    4.  各 `scaffold` に対して:
        *   `originalDir = filepath.Join(baseDir, scaffold.OriginalRef)`
        *   `zipPath = filepath.Join(baseDir, scaffold.TemplateRef + ".zip")`
        *   進捗ログ: `fmt.Printf("Archiving %s -> %s\n", originalDir, zipPath)`
        *   `archiver.ZipDirectory(originalDir, zipPath)` を呼び出し
        *   エラー発生時はエラーメッセージを出力して `os.Exit(1)`
    5.  全件成功時に完了メッセージを出力

---

### Go モジュール

#### [MODIFY] [go.mod](file://features/templatizer/go.mod)
*   **Description**: `gopkg.in/yaml.v3` 依存を追加
*   **Logic**:
    *   `go get gopkg.in/yaml.v3` を実行して依存を追加
    *   `go get github.com/stretchr/testify` を実行してテスト依存を追加

## Step-by-Step Implementation Guide

1.  **依存追加**:
    *   `features/templatizer/` で `go get gopkg.in/yaml.v3` と `go get github.com/stretchr/testify` を実行

2.  **catalog テスト作成** (TDD: テスト先行):
    *   `features/templatizer/internal/catalog/catalog_test.go` を作成
    *   テストケース: 正常系（3 scaffold）、空 scaffolds、不正 YAML
    *   この時点でテストが失敗することを確認（`ParseCatalog` 未実装）

3.  **catalog 実装**:
    *   `features/templatizer/internal/catalog/catalog.go` を作成
    *   `Scaffold`, `Catalog` 構造体を定義
    *   `ParseCatalog`, `LoadCatalog` を実装
    *   テストが通ることを確認

4.  **archiver テスト作成** (TDD: テスト先行):
    *   `features/templatizer/internal/archiver/archiver_test.go` を作成
    *   テストケース: 正常系（サブディレクトリ含む）、存在しないディレクトリ
    *   この時点でテストが失敗することを確認

5.  **archiver 実装**:
    *   `features/templatizer/internal/archiver/archiver.go` を作成
    *   `ZipDirectory` を実装
    *   テストが通ることを確認

6.  **main.go 実装**:
    *   既存の Hello World を CLI オーケストレーションに書き換え
    *   引数処理、catalog 読み込み、各 scaffold の ZIP 生成ループを実装

7.  **ビルド・テスト検証**:
    *   `./scripts/process/build.sh` を実行して全体ビルドと単体テストを通す

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **期待結果**: ビルド成功、catalog/archiver の全単体テストがパス
    *   **確認ポイント**:
        *   `features/templatizer` がビルドされて `bin/templatizer` が生成されること
        *   catalog の YAML 解析テスト 3 ケースがパス
        *   archiver の ZIP 圧縮テスト 2 ケースがパス

2.  **E2E 手動検証** (ビルド後):
    *   実際に `bin/templatizer catalog.yaml` を実行し、以下を確認:
        *   `catalog/templates/root/project-default.zip` が生成されること
        *   `catalog/templates/axsh/go-standard-project.zip` が生成されること
        *   `catalog/templates/axsh/go-standard-feature.zip` が生成されること
        *   各 ZIP を展開すると `catalog/originals/` 内の対応ディレクトリと同一内容であること

## Documentation

本計画で影響を受ける既存ドキュメントはありません。
