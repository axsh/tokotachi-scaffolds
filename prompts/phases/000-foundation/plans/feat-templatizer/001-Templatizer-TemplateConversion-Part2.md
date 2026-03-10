# 001-Templatizer-TemplateConversion-Part2

> **Source Specification**: [001-Templatizer-TemplateConversion.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/001-Templatizer-TemplateConversion.md)

## Goal Description

テンプレート変換パイプラインの**個別変換コンポーネント**を実装する：
1. AST 変換（Tree-Sitter による `go.mod` / `*.go` の構文安全な置換）
2. ディレクトリ・ファイルリネーム（`cmd/<旧名>` → `cmd/<新名>`）
3. ヒントファイル処理（`*.hints` による文字列置換 + `{{param}}` プレースホルダー展開）

## User Review Required

> [!WARNING]
> Tree-Sitter の Go バインディングとして `github.com/smacker/go-tree-sitter` の使用を想定しています。このライブラリは CGo に依存します。CGo が使えない環境では、代替として Go 標準の `go/parser` + `go/ast` パッケージの使用も検討可能です。`go.mod` パースは正規表現で十分安全に対応できます。どちらのアプローチを採用するか、ご確認ください。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R4: AST 解析によるコード変換 | Proposed Changes > converter/ast_transformer |
| R5: ディレクトリ・ファイル名のリネーム | Proposed Changes > converter/renamer |
| R6: ヒントファイルによる周辺ファイルの変換 | Proposed Changes > converter/hint_processor |
| R7: テンプレートエンジンとプレースホルダー構文 | Proposed Changes > converter/hint_processor |

## Proposed Changes

### converter パッケージ — AST 変換

#### [NEW] [ast_transformer_test.go](file://features/templatizer/internal/converter/ast_transformer_test.go)
*   **Description**: AST 変換の単体テスト（TDD: テスト先行）
*   **Technical Design**:
    ```go
    func TestTransformGoMod(t *testing.T) {
        tests := []struct {
            name      string
            input     string // go.mod の内容
            oldModule string
            newModule string
            want      string // 変換後の go.mod 内容
        }{...}
    }

    func TestTransformGoImports(t *testing.T) {
        tests := []struct {
            name      string
            input     string // .go ファイルの内容
            oldModule string
            newModule string
            want      string // 変換後の内容
        }{...}
    }
    ```
*   **Logic (テストケース)**:
    - **go.mod 変換**:
      1. `module function` → `module github.com/new-org/new-app` が正しく置換される
      2. `module github.com/old-org/old-app` → `module github.com/new-org/new-app`
      3. `require` などの他の行は変更されない
    - **import パス変換**:
      1. `import "function/internal/config"` → `import "github.com/new-org/new-app/internal/config"`
      2. 複数 import がある場合、旧モジュール名と前方一致するもののみ置換
      3. 外部パッケージ（`github.com/spf13/cobra` 等）は変更されない
      4. コメント内のモジュールパス文字列は変更されない
      5. リテラル文字列内のモジュールパス文字列は変更されない

#### [NEW] [ast_transformer.go](file://features/templatizer/internal/converter/ast_transformer.go)
*   **Description**: Go ソースコードの AST ベース変換（R4）
*   **Technical Design**:
    ```go
    package converter

    // TransformGoMod は go.mod ファイルの内容を変換する。
    // module ディレクティブの旧モジュール名を新モジュール名に置換する。
    // 返り値: 変換後の内容, 変換が行われたか, エラー
    func TransformGoMod(content []byte, oldModule, newModule string) ([]byte, bool, error)

    // TransformGoSource は .go ファイルの内容を変換する。
    // import パスの旧モジュール名を新モジュール名に置換する。
    // 返り値: 変換後の内容, 変換が行われたか, エラー
    func TransformGoSource(content []byte, oldModule, newModule string) ([]byte, bool, error)
    ```
*   **Logic**:
    - **`TransformGoMod`**:
      1. Go 標準 `modfile` パッケージ（`golang.org/x/mod/modfile`）で `go.mod` をパース
      2. `Module.Mod.Path` が `oldModule` と一致する場合、`newModule` に置換
      3. `modfile.Format` で再フォーマットして返す
      4. 一致しない場合、変換なし（`false`）を返す
    - **`TransformGoSource`**:
      1. Go 標準 `go/parser` + `go/ast` で `.go` ファイルをパース
      2. `ast.Inspect` で全 `ast.ImportSpec` を走査
      3. 各 import パスの値が `oldModule` と前方一致する場合、`oldModule` 部分を `newModule` に置換
      4. `go/format.Source` で再フォーマットして返す
      5. コメントやリテラル文字列は `ImportSpec` ノードではないため影響を受けない

> **設計判断**: 仕様書では Tree-Sitter を検討していたが、Go 標準ライブラリ (`go/parser`, `go/ast`, `golang.org/x/mod/modfile`) で十分安全な AST 変換が可能であり、CGo 依存を避けられるため、そちらを優先的に採用する。ユーザー確認後に決定。

#### [NEW] [transform.go](file://features/templatizer/internal/converter/transform.go)
*   **Description**: ディレクトリ全体の Go ファイルを走査して AST 変換を適用するヘルパー
*   **Technical Design**:
    ```go
    // TransformResult は変換結果を保持する
    type TransformResult struct {
        Path        string // 元のファイルパス
        Transformed bool   // 変換が行われたか
    }

    // TransformGoFiles は tempDir 内の全 .go ファイルと go.mod を走査し、
    // AST 変換を適用する。変換されたファイルには .tmpl ポストフィックスを付与する。
    // replacements の最初のエントリ（最長一致）を module 置換に使用する。
    func TransformGoFiles(tempDir string, oldModule, newModule string) ([]TransformResult, error)
    ```
*   **Logic**:
    1. `filepath.WalkDir` で tempDir を走査
    2. `go.mod` → `TransformGoMod` を適用、変換された場合 `go.mod.tmpl` にリネーム
    3. `*.go` → `TransformGoSource` を適用、変換された場合 `<name>.go.tmpl` にリネーム
    4. 変換結果のリストを返す

---

### converter パッケージ — リネーム

#### [NEW] [renamer_test.go](file://features/templatizer/internal/converter/renamer_test.go)
*   **Description**: ディレクトリリネーム処理の単体テスト（TDD）
*   **Logic (テストケース)**:
    1. `cmd/old-app/` が存在 → `cmd/new-app/` にリネームされる
    2. `cmd/old-app/main.go` の内容がリネーム後も保持される
    3. `cmd/` ディレクトリが存在しない → エラーにならない（スキップ）
    4. `cmd/<旧名>` が存在しない → エラーにならない（スキップ）

#### [NEW] [renamer.go](file://features/templatizer/internal/converter/renamer.go)
*   **Description**: ディレクトリ・ファイル名のリネーム（R5）
*   **Technical Design**:
    ```go
    // RenameDirectories は tempDir 内で旧名を新名にリネームする。
    // 現時点では cmd/<oldName> → cmd/<newName> のパターンのみ対応。
    func RenameDirectories(tempDir, oldName, newName string) error
    ```
*   **Logic**:
    1. `filepath.Join(tempDir, "cmd", oldName)` の存在を確認
    2. 存在する場合 `os.Rename` で `cmd/<newName>` にリネーム
    3. 存在しない場合はスキップ（エラーにしない）

---

### converter パッケージ — ヒントファイル処理

#### [NEW] [hint_processor_test.go](file://features/templatizer/internal/converter/hint_processor_test.go)
*   **Description**: ヒントファイル処理の単体テスト（TDD）
*   **Logic (テストケース)**:
    1. `Makefile` + `Makefile.hints` → `Makefile` 内の文字列が置換されること
    2. `replace_with` 内の `{{param}}` がパラメータ値に展開されること
    3. 処理後 `Makefile.hints` が削除されること
    4. 処理後、対象ファイルが `.tmpl` 付きにリネームされること（`Makefile` → `Makefile.tmpl`）
    5. ヒントファイルが存在しない場合 → エラーにならない
    6. 不正な YAML のヒントファイル → エラーを返す

#### [NEW] [hint_processor.go](file://features/templatizer/internal/converter/hint_processor.go)
*   **Description**: ヒントファイルによる文字列置換処理（R6, R7）
*   **Technical Design**:
    ```go
    // HintFile はヒントファイルのYAML構造
    type HintFile struct {
        Replacements []HintReplacement `yaml:"replacements"`
    }

    type HintReplacement struct {
        Match       string `yaml:"match"`
        ReplaceWith string `yaml:"replace_with"`
    }

    // ProcessHints は tempDir 内の全 *.hints ファイルを探索し、
    // 対応するファイルに置換ルールを適用する。
    // params はプレースホルダー展開用のパラメータマップ（name → value）。
    // 処理後、対象ファイルは .tmpl 付きにリネーム、.hints は削除される。
    func ProcessHints(tempDir string, params map[string]string) error
    ```
*   **Logic**:
    1. `filepath.WalkDir` で `*.hints` ファイルを探索
    2. 各 `.hints` ファイルを `yaml.Unmarshal` でパース
    3. 対象ファイル名 = ヒントファイル名から `.hints` を除去（例: `Makefile.hints` → `Makefile`）
    4. 対象ファイルの内容を読み込み
    5. 各 `Replacement` について:
       - `replace_with` 内の `{{param}}` を `params[param]` の値に展開（`strings.ReplaceAll`）
       - `match` の文字列を展開済み `replace_with` で置換（`strings.ReplaceAll`）
    6. 変換後の内容を対象ファイルに書き戻す
    7. 対象ファイルを `<name>.tmpl` にリネーム
    8. `.hints` ファイルを `os.Remove` で削除

## Step-by-Step Implementation Guide

1. **[ ] AST 変換テスト作成**:
    *   `ast_transformer_test.go` に `TestTransformGoMod` と `TestTransformGoImports` を記述
    *   テストが FAIL することを確認（TDD）

2. **[ ] AST 変換実装**:
    *   `ast_transformer.go` に `TransformGoMod` と `TransformGoSource` を実装
    *   `golang.org/x/mod/modfile` を `go.mod` に追加
    *   テストが PASS することを確認

3. **[ ] ディレクトリ走査＋.tmpl付与ヘルパー実装**:
    *   `transform.go` に `TransformGoFiles` を実装

4. **[ ] リネームテスト作成**:
    *   `renamer_test.go` にテストを記述、FAIL 確認（TDD）

5. **[ ] リネーム実装**:
    *   `renamer.go` に `RenameDirectories` を実装、テスト PASS 確認

6. **[ ] ヒントファイル処理テスト作成**:
    *   `hint_processor_test.go` にテストを記述、FAIL 確認（TDD）

7. **[ ] ヒントファイル処理実装**:
    *   `hint_processor.go` に `ProcessHints` を実装、テスト PASS 確認

8. **[ ] ビルドパイプライン実行**:
    *   `scripts/process/build.sh` で全テスト通過を確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    * `ast_transformer_test.go`, `renamer_test.go`, `hint_processor_test.go` が全て PASS すること

## Documentation

None.

## 継続計画について

本計画は Part2 です。
- **Part1** (前): catalog 拡張 + クリーンアップ + originals リファイン + build.sh 拡張
- **Part3** (次): 変換パイプライン統合（`converter.go`）+ `main.go` 変更
