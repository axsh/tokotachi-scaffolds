# 008-GoMod-Templatize-Fix

> **Source Specification**: `prompts/phases/000-foundation/ideas/feat-templatizer/008-GoMod-Templatize-Fix.md`

## Goal Description

`TransformGoMod` 関数を修正し、`go.mod` の `module` 行を無条件にテンプレート変数に置換するようにする。現在は `oldModule` との完全一致が必要で、originals の `go.mod` モジュールパスと一致しないためテンプレート化がスキップされる問題を修正する。

## User Review Required

> [!IMPORTANT]
> `TransformGoMod` のシグネチャ変更により、`oldModule` パラメータを削除します。代わりに `go.mod` から実際のモジュールパスを返すようにし、`.go` ファイルの import 変換に活用します。これによりパイプライン全体の `oldModule` の決定方法が変更されます。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| go.mod の module 行を自動的にテンプレート化する | Proposed Changes > `ast_transformer.go` の `TransformGoMod` 修正 |
| テンプレート化のロジック変更 (exact match 廃止) | Proposed Changes > `ast_transformer.go` の `TransformGoMod` 修正 |
| newModule の構成 (`{{module_path}}/{{program_name}}`) | Proposed Changes > `converter.go` の `BuildConvertParams` 修正 |
| 既存テストの更新 | Proposed Changes > `ast_transformer_test.go`, `converter_test.go` 修正 |
| go.mod テンプレート化のテストケース追加 | Proposed Changes > `ast_transformer_test.go`, `converter_test.go` 修正 |

## Proposed Changes

### converter パッケージ

#### [MODIFY] [ast_transformer_test.go](file://features/templatizer/internal/converter/ast_transformer_test.go)
*   **Description**: `TestTransformGoMod` のテストケースを新シグネチャに合わせて更新し、新テストケースを追加
*   **Technical Design**:
    *   テーブル駆動テストの構造体を変更:
        ```go
        tests := []struct {
            name              string
            input             string
            newModule         string
            wantOutput        string
            wantOrigModule    string  // 新: go.mod から取得した元のモジュールパス
            wantChanged       bool
        }
        ```
    *   呼び出し箇所: `TransformGoMod([]byte(tt.input), tt.newModule)` (引数2つに変更)
    *   返り値検証: `transformed, origModule, changed, err` を全て検証
*   **Logic**:
    *   既存テスト4ケースを新シグネチャに変換:
        1. `"replaces simple module name"`: `newModule="github.com/new-org/new-app"`, `wantOrigModule="function"`, `wantChanged=true`
        2. `"replaces full module path"`: `newModule="github.com/new-org/new-app"`, `wantOrigModule="github.com/old-org/old-app"`, `wantChanged=true`
        3. `"preserves require block"`: require ブロックが残ることを確認, `wantChanged=true`
        4. `"no change when module does not match"` → `"always replaces module line"` に変更。oldModule のマッチ不要になったため、**常に変換される**。`wantChanged=true` に変更
    *   新テストケースを追加:
        5. `"replaces long module path (real-world scaffold case)"`: `input="module github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature\n\ngo 1.24.0\n"`, `newModule="{{module_path}}/{{program_name}}"`, `wantOutput="module {{module_path}}/{{program_name}}\n\ngo 1.24.0\n"`, `wantOrigModule="github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature"`, `wantChanged=true`
        6. `"no module line returns unchanged"`: module 行がない `go.mod` の場合、`wantChanged=false`, `wantOrigModule=""`

#### [MODIFY] [ast_transformer.go](file://features/templatizer/internal/converter/ast_transformer.go)
*   **Description**: `TransformGoMod` 関数のシグネチャと実装を変更
*   **Technical Design**:
    ```go
    // TransformGoMod transforms the module directive in a go.mod file.
    // It replaces the module path with newModule unconditionally.
    // Returns: transformed content, original module path, whether changed, error.
    func TransformGoMod(content []byte, newModule string) ([]byte, string, bool, error)
    ```
*   **Logic**:
    1. `goModModuleRe` で module 行をマッチ
    2. マッチしなければ `(content, "", false, nil)` を返す
    3. マッチしたら `currentModule = strings.TrimSpace(match[1])` で元のモジュールパスを取得
    4. `currentModule == newModule` の場合は変更なし: `(content, currentModule, false, nil)` を返す
    5. それ以外は `goModModuleRe.ReplaceAllStringFunc` で `"module " + newModule` に置換
    6. `([]byte(result), currentModule, true, nil)` を返す
    7. **変更点**: `oldModule` パラメータ削除、`oldModule` との比較判定を削除、返り値に `originalModule string` 追加

#### [MODIFY] [transform.go](file://features/templatizer/internal/converter/transform.go)
*   **Description**: `TransformGoFiles` 内の `TransformGoMod` 呼び出しを新シグネチャに合わせ、`go.mod` から取得したモジュールパスを `.go` ファイルの変換に利用する
*   **Technical Design**:
    ```go
    // TransformGoFiles walks tempDir and applies AST transformations.
    // go.mod is processed first to discover the original module path,
    // which is then used for .go file import transformations.
    func TransformGoFiles(tempDir string, oldModule, newModule string) ([]TransformResult, error)
    ```
    シグネチャは変更しない（外部からの呼び出し互換性維持）。ただし内部ロジックを変更。
*   **Logic**:
    1. 2パス方式に変更:
        - **Pass 1**: `go.mod` ファイルのみを処理。`TransformGoMod(content, newModule)` を呼び出し、返り値の `originalModule` を保存
        - **Pass 2**: `.go` ファイルを処理。`TransformGoSource(content, effectiveOldModule, newModule)` を呼び出し。`effectiveOldModule` は Pass 1 で取得した `originalModule`（取得できなかった場合は引数の `oldModule` にフォールバック）
    2. 具体的な実装:
        ```go
        // 2パスでwalkする方法ではなく、1パスで実装する
        // まず go.mod を先に探してモジュールパスを取得、その後 .go ファイルを処理
        
        // Step 1: Find and process go.mod first
        var discoveredModule string
        // filepath.WalkDir で go.mod を探し、TransformGoMod を呼び出す
        // discoveredModule に結果を保存
        
        // Step 2: Determine effective oldModule
        effectiveOldModule := oldModule
        if discoveredModule != "" {
            effectiveOldModule = discoveredModule
        }
        
        // Step 3: Process .go files using effectiveOldModule
        // filepath.WalkDir で .go ファイルを処理
        ```
    3. 実装上は2回 `WalkDir` する必要があるが、テンポラリディレクトリはファイル数が少ないため性能問題はない

#### [MODIFY] [converter.go](file://features/templatizer/internal/converter/converter.go)
*   **Description**: `BuildConvertParams` での `NewModule` の値を `{{module_path}}/{{program_name}}` 形式のテンプレート変数に変更
*   **Technical Design**:
    `BuildConvertParams` 内の `case "module_path"` ブロックで `NewModule` をテンプレート変数形式に設定:
    ```go
    case "module_path":
        params.OldModule = oldValue
        // NewModule will be set after all params are processed
    case "program_name":
        params.OldProgram = oldValue
        params.NewProgram = oldValue
    ```
    ループ終了後:
    ```go
    // Construct template variable for module path
    // format: "{{module_path}}/{{program_name}}" → becomes the newModule in go.mod
    if _, hasModulePath := params.HintParams["module_path"]; hasModulePath {
        if _, hasProgramName := params.HintParams["program_name"]; hasProgramName {
            params.NewModule = "{{module_path}}/{{program_name}}"
        } else {
            params.NewModule = "{{module_path}}"
        }
    }
    ```
*   **Logic**:
    *   `NewModule` は実際のモジュールパスではなく、テンプレート変数の文字列を設定する
    *   `OldModule` は引き続き `old_value` → `default` フォールバックで設定（`.go` ファイルのフォールバック用）
    *   `program_name` が定義されていない場合は `{{module_path}}` のみ

#### [MODIFY] [converter_test.go](file://features/templatizer/internal/converter/converter_test.go)
*   **Description**: 全体パイプラインテスト `TestConvert` を更新し、実際のユースケースを反映するテストケースを追加
*   **Technical Design**:
    *   既存の `"full pipeline execution"` テスト:
        - `go.mod` の `module old-org/old-app` → テンプレート変数への変換を確認
        - `goModContent` のアサーション変更: `module github.com/new-org/new-app` → `module {{module_path}}/{{program_name}}`
        - `ConvertParams` に `NewModule: "{{module_path}}/{{program_name}}"` を設定
    *   新テストケース `"real-world scaffold: go.mod module mismatch with oldModule"`:
        - `go.mod` のモジュール (`github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature`) と `OldModule` (`github.com/axsh/tokotachi/features/myprog`) が不一致
        - テンプレート化が成功し、`go.mod.tmpl` に `module {{module_path}}/{{program_name}}` が含まれることを確認
        - `.go` ファイルの import が `go.mod` から取得した `effectiveOldModule` で正しく変換されることを確認
*   **Logic**:
    *   新テストケースの `go.mod`:
        ```
        module github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature
        
        go 1.24.0
        ```
    *   新テストケースの `main.go`:
        ```go
        package main
        
        import "github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature/internal/pkg"
        
        func main() {}
        ```
    *   `ConvertParams`:
        ```go
        params := ConvertParams{
            OldModule:  "github.com/axsh/tokotachi/features/myprog", // scaffold.yaml default
            NewModule:  "{{module_path}}/{{program_name}}",
            OldProgram: "myprog",
            NewProgram: "myprog",
            HintParams: map[string]string{...},
        }
        ```
    *   期待結果:
        - `go.mod.tmpl` が存在し `module {{module_path}}/{{program_name}}` を含む
        - `main.go.tmpl` が存在し `{{module_path}}/{{program_name}}/internal/pkg` を含む

#### [MODIFY] [converter_test.go](file://features/templatizer/internal/converter/converter_test.go) - `TestBuildConvertParamsOldValueFallback`
*   **Description**: `BuildConvertParams` が `NewModule` をテンプレート変数形式で返すことを検証するテストを追加
*   **Logic**:
    *   既存テスト `"falls back to default when old_value is empty"` に `NewModule` アサーション追加:
        ```go
        if params.NewModule != "{{module_path}}/{{program_name}}" {
            t.Errorf(...)
        }
        ```
    *   既存テスト `"explicit old_value takes priority over default"` にも同様のアサーション追加

## Step-by-Step Implementation Guide

- [/] 1. **`ast_transformer_test.go` のテスト更新 (TDD: Red)**
    - [ ] `TestTransformGoMod` のテーブル構造体に `wantOrigModule string` フィールドを追加
    - [ ] 全テストケースから `oldModule` フィールドを削除
    - [ ] 呼び出しを `TransformGoMod([]byte(tt.input), tt.newModule)` に変更
    - [ ] 返り値検証を `transformed, origModule, changed, err` に変更
    - [ ] `origModule` のアサーションを追加
    - [ ] 既存ケース4を `"always replaces module line"` に変更 (`wantChanged=true`)
    - [ ] 新ケース5 `"replaces long module path (real-world scaffold case)"` を追加
    - [ ] 新ケース6 `"no module line returns unchanged"` を追加
    - [ ] → **ビルド失敗を確認** (シグネチャ不一致)

- [ ] 2. **`ast_transformer.go` の実装 (TDD: Green)**
    - [ ] `TransformGoMod` のシグネチャを `(content []byte, newModule string) ([]byte, string, bool, error)` に変更
    - [ ] `oldModule` パラメータと比較ロジックを削除
    - [ ] 元のモジュールパスを返すロジックを追加
    - [ ] `currentModule == newModule` の場合は未変更で返すガード追加
    - [ ] → **`ast_transformer_test.go` が全てパスすることを確認**

- [ ] 3. **`transform.go` の実装**
    - [ ] `TransformGoFiles` 内を2パス方式に変更
    - [ ] Pass 1: `go.mod` を探して `TransformGoMod(content, newModule)` を呼び出し、`discoveredModule` を保存
    - [ ] Pass 2: `.go` ファイルを `effectiveOldModule` (`discoveredModule` → `oldModule` フォールバック) で変換
    - [ ] → **ビルド成功を確認**

- [ ] 4. **`converter.go` の `BuildConvertParams` 修正**
    - [ ] `NewModule` の設定を `"{{module_path}}/{{program_name}}"` テンプレート変数形式に変更
    - [ ] `module_path` のみの場合は `"{{module_path}}"` に設定

- [ ] 5. **`converter_test.go` のテスト更新**
    - [ ] 既存 `"full pipeline execution"` の `ConvertParams` を更新: `NewModule: "{{module_path}}/{{program_name}}"`
    - [ ] `goModContent` のアサーションを `module {{module_path}}/{{program_name}}` に変更
    - [ ] 新テストケース `"real-world scaffold: go.mod module mismatch with oldModule"` を追加
    - [ ] `TestBuildConvertParamsOldValueFallback` に `NewModule` アサーションを追加
    - [ ] → **全テストパスを確認**

- [ ] 6. **ビルド & テスト検証**
    - [ ] `scripts/process/build.sh` 実行
    - [ ] `scripts/process/integration_test.sh` 実行（該当テストがあれば）

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `TestTransformGoMod` の全ケース（6ケース）がPASSすること
    *   `TestTransformGoSource` が変更なしでPASSすること
    *   `TestConvert` の全ケース（3ケース: 既存2 + 新規1）がPASSすること
    *   `TestBuildConvertParamsOldValueFallback` が更新後もPASSすること

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   既存の統合テストにリグレッションがないこと

## Documentation

本変更は内部実装の修正であり、既存ドキュメントへの影響はありません。
