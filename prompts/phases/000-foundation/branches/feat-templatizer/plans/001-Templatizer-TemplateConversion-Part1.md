# 001-Templatizer-TemplateConversion-Part1

> **Source Specification**: [001-Templatizer-TemplateConversion.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/001-Templatizer-TemplateConversion.md)

## Goal Description

テンプレート変換システムの**基盤部分**を構築する。具体的には：
1. `catalog.yaml` の `template_params` をパースできるよう `Scaffold` 構造体を拡張する
2. クリーンアップ処理（不要ファイル除外）を実装する
3. 既存 originals テンプレート (`go-standard-feature`) をビルド可能な状態にリファインする
4. `build.sh` を originals 配下の Go プロジェクトもビルド・テストできるよう拡張する

## User Review Required

> [!IMPORTANT]
> `go-standard-feature` の go.mod モジュール名を `github.com/axsh/tokotachi/features/myprog` とする想定です。この値は `catalog.yaml` の `template_params[].old_value` と一致します。問題があればお知らせください。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: テンプレート・パラメータ（catalog.yaml定義） | Proposed Changes > catalog パッケージ |
| R2: クリーンアップ（不要ファイルの除外） | Proposed Changes > converter/cleaner |
| R9: 既存 originals リファインメント (go-standard-feature) | Proposed Changes > originals リファイン |
| R10: build.sh の originals ビルド対応 | Proposed Changes > build.sh |

> R9 の `go-kotoshiro-mcp-feature` は既にユーザーにより go.mod 等が追加済みのため、本計画では対象外。

## Proposed Changes

### catalog パッケージ

#### [NEW] [cleaner_test.go](file://features/templatizer/internal/converter/cleaner_test.go)
*   **Description**: クリーンアップ処理の単体テスト（TDD: テスト先行）
*   **Technical Design**:
    ```go
    // テーブル駆動テスト
    func TestClean(t *testing.T) {
        tests := []struct {
            name     string
            setup    func(dir string) // テスト用ファイル構造を作成
            wantGone []string         // 削除されるべきパス
            wantKeep []string         // 残るべきパス
        }{...}
    }
    ```
*   **Logic (テストケース)**:
    1. `.git/` ディレクトリ → 削除されること
    2. `go.sum` → 削除されること
    3. `vendor/` → 削除されること
    4. `bin/` → 削除されること
    5. `.DS_Store` → 削除されること
    6. `main.go`, `go.mod`, `internal/` → 残ること
    7. 除外対象が存在しない場合 → エラーにならないこと

#### [NEW] [cleaner.go](file://features/templatizer/internal/converter/cleaner.go)
*   **Description**: クリーンアップ処理（R2）
*   **Technical Design**:
    ```go
    package converter

    // DefaultExcludes はクリーンアップ時に除外するパターン一覧
    var DefaultExcludes = []string{
        ".git",
        "go.sum",
        "vendor",
        "bin",
        ".DS_Store",
    }

    // Clean は tempDir 内の不要ファイル・ディレクトリを削除する。
    // excludes に含まれるファイル名・ディレクトリ名と一致するエントリを
    // tempDir 直下・再帰的に探索して os.RemoveAll で削除する。
    func Clean(tempDir string, excludes []string) error
    ```
*   **Logic**:
    1. `filepath.WalkDir` で tempDir を走査
    2. 各エントリの `filepath.Base()` が `excludes` のいずれかと一致するか判定
    3. 一致した場合 `os.RemoveAll` で削除し、ディレクトリの場合は `fs.SkipDir` を返す
    4. 一致しない場合はスキップ

#### [MODIFY] [catalog.go](file://features/templatizer/internal/catalog/catalog.go)
*   **Description**: `Scaffold` 構造体に `TemplateParams` フィールドを追加（R1）
*   **Technical Design**:
    ```go
    // TemplateParam はテンプレート変換パラメータの1エントリ
    type TemplateParam struct {
        Name        string `yaml:"name"`
        Description string `yaml:"description"`
        Required    bool   `yaml:"required"`
        Default     string `yaml:"default,omitempty"`
        OldValue    string `yaml:"old_value"`
    }

    // Scaffold（既存構造体にフィールド追加）
    type Scaffold struct {
        Name           string          `yaml:"name"`
        Category       string          `yaml:"category"`
        Description    string          `yaml:"description"`
        TemplateRef    string          `yaml:"template_ref"`
        OriginalRef    string          `yaml:"original_ref"`
        TemplateParams []TemplateParam `yaml:"template_params,omitempty"`
    }
    ```
*   **Logic**: YAML タグにより `gopkg.in/yaml.v3` が自動的にパースする。追加コードは不要。

#### [MODIFY] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)
*   **Description**: `TemplateParams` パースの検証テストを追加
*   **Logic (テストケース追加)**:
    1. `template_params` を含む YAML を `ParseCatalog` に渡す
    2. パース結果の `TemplateParams` が正しい `Name`, `OldValue`, `Required` を持つことを検証
    3. `template_params` がない YAML → `TemplateParams` は `nil` または空スライス

---

### originals リファイン

#### [MODIFY] [go.mod.tmpl → go.mod](file://catalog/originals/axsh/go-standard-feature/base/go.mod.tmpl)
*   **Description**: `go.mod.tmpl`（Go テンプレート構文）を、ビルド可能な `go.mod`（実体）に置き換える
*   **Logic**:
    - ファイル名: `go.mod.tmpl` → `go.mod` にリネーム
    - 内容:
      ```
      module github.com/axsh/tokotachi/features/myprog

      go 1.24.0
      ```

---

### build.sh

#### [MODIFY] [build.sh](file://scripts/process/build.sh)
*   **Description**: `catalog/originals/` 配下の Go プロジェクト（`go.mod` を含むディレクトリ）もビルド・テスト対象にする（R10）
*   **Technical Design**:
    ```bash
    # 新しい関数を追加
    build_originals() {
        step "Originals: Build & Unit Test"
        cd "$PROJECT_ROOT"

        local found_any=false
        # find で catalog/originals 配下の go.mod を再帰的に探索
        while IFS= read -r gomod_path; do
            ...
        done < <(find catalog/originals -name "go.mod" -type f 2>/dev/null)
    }
    ```
*   **Logic**:
    1. `find catalog/originals -name "go.mod" -type f` で全 `go.mod` を探索
    2. 各 `go.mod` のディレクトリで:
       - `go test ./...`（`integration/` を除外: `grep -v '/integration'`）を実行
       - `go build ./...` を実行（バイナリ出力なし、コンパイル確認のみ）
    3. features のビルドとは別ステップ `"Originals: Build & Unit Test"` として表示
    4. `main()` 内で `build_go` の後に `build_originals` を呼び出す

## Step-by-Step Implementation Guide

1. **[ ] クリーンアップテスト作成**:
    *   `features/templatizer/internal/converter/` ディレクトリを作成
    *   `cleaner_test.go` にテーブル駆動テストを記述
    *   テストが FAIL することを確認（TDD）

2. **[ ] クリーンアップ実装**:
    *   `cleaner.go` に `Clean` 関数を実装
    *   テストが PASS することを確認

3. **[ ] catalog 構造体拡張テスト**:
    *   `catalog_test.go` に `TemplateParams` パーステストを追加
    *   テストが FAIL することを確認（TDD）

4. **[ ] catalog 構造体拡張**:
    *   `catalog.go` に `TemplateParam` 構造体と `Scaffold.TemplateParams` フィールドを追加
    *   テストが PASS することを確認

5. **[ ] go-standard-feature リファイン**:
    *   `go.mod.tmpl` を削除し、`go.mod` を作成（中身: `module github.com/axsh/tokotachi/features/myprog`）

6. **[ ] build.sh 拡張**:
    *   `build_originals()` 関数を追加
    *   `main()` に `build_originals` の呼び出しを追加

7. **[ ] ビルドパイプライン実行で全体検証**:
    *   `scripts/process/build.sh` を実行し、features と originals の両方がビルド・テスト通過することを確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    * templatizer の単体テスト（`cleaner_test.go`, `catalog_test.go`）が PASS すること
    * originals (`go-standard-feature/base`, `go-kotoshiro-mcp-feature/base`) のビルドが成功すること

## Documentation

None. 仕様書自体がドキュメントとして最新状態。

## 継続計画について

本計画は Part1 です。以下の Part が続きます:
- **Part2**: AST 変換（Tree-Sitter）+ ディレクトリリネーム + ヒントファイル処理（R4, R5, R6, R7）
- **Part3**: 変換パイプライン統合 + main.go 変更 + .tmpl 付与（R3, R8）
