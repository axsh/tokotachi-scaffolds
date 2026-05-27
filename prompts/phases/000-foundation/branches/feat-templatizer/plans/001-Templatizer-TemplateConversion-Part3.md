# 001-Templatizer-TemplateConversion-Part3

> **Source Specification**: [001-Templatizer-TemplateConversion.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/001-Templatizer-TemplateConversion.md)

## Goal Description

Part1（基盤）と Part2（個別コンポーネント）で作成した各モジュールを統合し、**テンプレート変換パイプライン**として機能させる：
1. `converter.go` でパイプライン制御（Step1〜4 の順序実行）
2. `main.go` でコピー後・ZIP前にパイプラインを呼び出す
3. `ConvertParams` の構築ロジック（`catalog.yaml` の `template_params` → `Replacements` マップ変換）

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: テンプレート・パラメータ（key-value マップ） | Proposed Changes > converter.go (`ConvertParams`) |
| R3: .tmpl ポストフィックス規約（templatizer が付与） | Proposed Changes > converter.go (パイプライン内で .tmpl 付与) |
| R8: 処理順序 (Step1→4) | Proposed Changes > converter.go |
| R1/R3/R8 の main.go 統合 | Proposed Changes > main.go |

## Proposed Changes

### converter パッケージ — パイプライン統合

#### [NEW] [converter_test.go](file://features/templatizer/internal/converter/converter_test.go)
*   **Description**: 変換パイプライン全体の統合テスト（TDD）
*   **Technical Design**:
    ```go
    func TestConvert(t *testing.T) {
        // テスト用のディレクトリ構造:
        // temp/
        //   go.mod         (module old-org/old-app)
        //   cmd/old-app/main.go (import "old-org/old-app/internal/pkg")
        //   internal/pkg/pkg.go
        //   .git/config
        //   go.sum
        //   Makefile
        //   Makefile.hints
    }
    ```
*   **Logic (テストケース)**:
    1. **パイプライン全体実行**: 上記構造を用意し `Convert` を実行
       - `.git/`, `go.sum` が削除されている（Step1: クリーンアップ）
       - `go.mod.tmpl` が生成されている（Step2: AST変換 + .tmpl付与）
       - `main.go.tmpl` が生成されている（Step2: import 変換 + .tmpl付与）
       - `internal/pkg/pkg.go` は `.tmpl` なし（変換不要のファイル）
       - `cmd/old-app/` が `cmd/new-app/` にリネームされている（Step3: リネーム）
       - `Makefile.tmpl` が生成されている（Step4: ヒントファイル適用 + .tmpl付与）
       - `Makefile.hints` が削除されている（Step4: ヒントファイル削除）
    2. **template_params なしの scaffold**: `Convert` がスキップ（何も変換しない）されること

#### [NEW] [converter.go](file://features/templatizer/internal/converter/converter.go)
*   **Description**: 変換パイプラインの制御（R3, R8）
*   **Technical Design**:
    ```go
    package converter

    // ConvertParams は変換パラメータを保持する。
    type ConvertParams struct {
        // Replacements は旧値→新値の対応表
        Replacements map[string]string
    }

    // Convert は tempDir に対してテンプレート変換パイプラインを実行する。
    // 処理順序:
    //   Step1: クリーンアップ
    //   Step2: AST変換（go.mod, *.go → .tmpl付与）
    //   Step3: ディレクトリリネーム
    //   Step4: ヒントファイル変換（*.hints → .tmpl付与）
    func Convert(tempDir string, params ConvertParams) error

    // BuildConvertParams は catalog の TemplateParams から ConvertParams を構築する。
    // 各 TemplateParam の old_value をキー、ユーザー入力値（または default）を値として
    // Replacements マップを構築する。
    func BuildConvertParams(templateParams []catalog.TemplateParam) ConvertParams
    ```
*   **Logic**:
    - **`Convert`**:
      1. `Clean(tempDir, DefaultExcludes)` — Step1
      2. Replacements から最初のエントリ（最長キー）を module 置換ペアとして使用
      3. `TransformGoFiles(tempDir, oldModule, newModule)` — Step2
      4. Replacements から `program_name` 相当のエントリを取得
      5. `RenameDirectories(tempDir, oldProgramName, newProgramName)` — Step3
      6. `ProcessHints(tempDir, paramsMap)` — Step4
    - **`BuildConvertParams`**:
      1. `catalog.TemplateParam` の配列をイテレート
      2. 各要素の `OldValue` → `Name` をキーバリューとして `Replacements` に追加
      3. 現時点では `OldValue` をそのまま使用（展開時にユーザー入力値で上書きされる設計）

---

### main.go — パイプライン統合

#### [MODIFY] [main.go](file://features/templatizer/main.go)
*   **Description**: コピー後・ZIP前に変換パイプラインを呼び出す
*   **Technical Design**:
    ```go
    func processScaffold(baseDir string, s catalog.Scaffold) error {
        // ... 既存コード（tempDir 作成, コピー）...

        // 新規: テンプレート変換パイプライン
        if len(s.TemplateParams) > 0 {
            params := converter.BuildConvertParams(s.TemplateParams)
            if err := converter.Convert(tempDir, params); err != nil {
                return fmt.Errorf("template conversion failed: %w", err)
            }
        }

        // ... 既存コード（ZIP 圧縮）...
    }
    ```
*   **Logic**:
    1. `s.TemplateParams` が空でない場合のみ変換パイプラインを実行
    2. `BuildConvertParams` で `ConvertParams` を構築
    3. `converter.Convert(tempDir, params)` を呼び出す
    4. エラー発生時は即座に返す（defer による tempDir クリーンアップは維持）

## Step-by-Step Implementation Guide

1. **[ ] パイプラインテスト作成**:
    *   `converter_test.go` にテスト用ディレクトリ構造のセットアップとパイプラインテストを記述
    *   テストが FAIL することを確認（TDD）

2. **[ ] ConvertParams と BuildConvertParams 実装**:
    *   `converter.go` に `ConvertParams` 構造体と `BuildConvertParams` 関数を実装

3. **[ ] Convert パイプライン実装**:
    *   `converter.go` に `Convert` 関数を実装（Step1〜4 の順序呼び出し）
    *   テストが PASS することを確認

4. **[ ] main.go 変更**:
    *   `processScaffold` 関数にパイプライン呼び出しを追加
    *   `converter` パッケージの import を追加

5. **[ ] 全体ビルドパイプライン実行**:
    *   `scripts/process/build.sh` で全テスト通過を確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    * 全ての単体テスト（Part1〜3）が PASS すること
    * originals のビルドが成功すること

## Documentation

None.
