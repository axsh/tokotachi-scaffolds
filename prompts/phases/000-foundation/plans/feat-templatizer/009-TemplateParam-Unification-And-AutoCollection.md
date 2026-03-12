# 009-TemplateParam-Unification-And-AutoCollection

> **Source Specification**: `prompts/phases/000-foundation/ideas/feat-templatizer/009-TemplateParam-Unification-And-AutoCollection.md`

## Goal Description

`program_name` を `feature_name` に統一し、テンプレートパラメータの自動蓄積機能を導入する。変換パイプラインの各ステップで発見されたテンプレート変数を `ParamCollector` で収集し、`scaffold.yaml` の定義とマージしてシャードYAMLの `template_params` に反映する。

## User Review Required

> [!IMPORTANT]
> `program_name` → `feature_name` の変更は **全ての originals の scaffold.yaml**、**converter コード**、**全テスト** に影響します。この変更は 008 で導入した `{{module_path}}/{{program_name}}` を `{{module_path}}/{{feature_name}}` に変更することも含みます。

> [!IMPORTANT]
> パラメータ自動蓄積の設計: `base_dir` のテンプレート変数を含め、変換中に発見されたパラメータを `scaffold.yaml` の `template_params` とマージする方針です。`scaffold.yaml` に未定義のパラメータが発見された場合は警告を出力し自動追加します。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `program_name` → `feature_name` 統一 | scaffold.yaml × 2, `converter.go`, `converter_test.go`, `ast_transformer_test.go`, `hint_processor_test.go` |
| テンプレートパラメータ自動蓄積 | 新規 `param_collector.go`, `param_collector_test.go` |
| `base_dir` からのパラメータ抽出 | `param_collector.go` の `ExtractParamsFromBaseDir` |
| シャードYAMLへの反映 | `main.go` の `processScaffold` + `generateShardFiles` 修正 |
| 未定義パラメータ検出 | `param_collector.go` の `MergeParams` |

## Proposed Changes

### originals scaffold.yaml

#### [MODIFY] [go-standard-feature/scaffold.yaml](file://catalog/originals/axsh/go-standard-feature/scaffold.yaml)
*   **Description**: `program_name` → `feature_name` にリネーム
*   **Logic**:
    ```yaml
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        default: "github.com/axsh/tokotachi/features/myprog"
      - name: "feature_name"         # ← program_name から変更
        description: "Feature name"   # ← description も更新
        required: true
        default: "myprog"
    ```

#### [MODIFY] [go-kotoshiro-mcp-feature/scaffold.yaml](file://catalog/originals/axsh/go-kotoshiro-mcp-feature/scaffold.yaml)
*   **Description**: `program_name` → `feature_name` にリネーム
*   **Logic**: 上記と同様に `program_name` → `feature_name`, description → `"Feature name"`

---

### converter パッケージ

#### [NEW] [param_collector_test.go](file://features/templatizer/internal/converter/param_collector_test.go)
*   **Description**: パラメータ収集と、scaffold.yaml 定義とのマージのテスト
*   **Technical Design**:
    ```go
    func TestExtractTemplateVars(t *testing.T)
    // テーブル駆動テスト:
    // - "{{module_path}}/{{feature_name}}" → ["feature_name", "module_path"]
    // - "features/{{feature_name}}" → ["feature_name"]
    // - "no-template-vars" → []
    // - "{{a}}/{{b}}/{{a}}" → ["a", "b"] (重複除去)

    func TestParamCollector(t *testing.T)
    // - Add で変数を追加
    // - Names() でソート済みリストを取得
    // - 重複追加は無視される

    func TestMergeParams(t *testing.T)
    // テーブル駆動テスト:
    // - scaffold.yaml 定義と完全一致 → そのまま返す
    // - 収集結果に未定義パラメータあり → 自動追加（name のみ、required=true）
    // - scaffold.yaml に定義されているが収集されなかったパラメータ → そのまま残す
    ```

#### [NEW] [param_collector.go](file://features/templatizer/internal/converter/param_collector.go)
*   **Description**: テンプレート変数の収集とマージ機能
*   **Technical Design**:
    ```go
    // ParamCollector accumulates template variable names found during conversion.
    type ParamCollector struct {
        params map[string]bool // set of discovered param names
    }

    // NewParamCollector creates a new ParamCollector.
    func NewParamCollector() *ParamCollector

    // Add records a discovered template variable name.
    func (pc *ParamCollector) Add(name string)

    // Names returns sorted list of all discovered param names.
    func (pc *ParamCollector) Names() []string

    // ExtractTemplateVars extracts {{xxx}} variable names from a string.
    // Returns sorted deduplicated list.
    func ExtractTemplateVars(s string) []string

    // MergeParams merges discovered params (from ParamCollector) with
    // the original scaffold.yaml template_params.
    // - Params in both: use scaffold.yaml definition (has description, default, etc.)
    // - Params discovered but not in scaffold.yaml: auto-add with name only, required=true
    // - Params in scaffold.yaml but not discovered: keep as-is
    func MergeParams(defined []catalog.TemplateParam, discovered []string) []catalog.TemplateParam
    ```
*   **Logic**:
    *   `ExtractTemplateVars`: 正規表現 `\{\{(\w+)\}\}` で `{{xxx}}` をマッチ、キャプチャグループ1を取得、`sort.Strings` でソート、重複除去
    *   `MergeParams`:
        1. `defined` を map[name]→TemplateParam に変換
        2. `discovered` を走査: defined に存在すれば result に追加、存在しなければ `TemplateParam{Name: name, Required: true}` を result に追加し stderr に警告出力
        3. `defined` のうち discovered に含まれないものも result に追加（既存パラメータを消さない）
        4. result を Name でソートして返す

#### [MODIFY] [converter.go](file://features/templatizer/internal/converter/converter.go)
*   **Description**: `program_name` → `feature_name` 変更と `Convert` にパラメータ収集を統合
*   **Technical Design**:
    *   `ConvertParams`:
        ```go
        type ConvertParams struct {
            OldModule  string
            NewModule  string
            OldProgram string            // ← コメント更新: "OldFeatureName" にリネーム検討
            NewProgram string            // ← 同上
            HintParams map[string]string
        }
        ```
        > `OldProgram`/`NewProgram` フィールド名は `RenameDirectories` で `cmd/<old>` → `cmd/<new>` のリネームに使用されているため、フィールド名自体は維持する。ただしコメントを更新する。
    *   `Convert` の返り値に `*ParamCollector` を追加:
        ```go
        func Convert(tempDir string, params ConvertParams) (*ParamCollector, error)
        ```
    *   `BuildConvertParams`: `program_name` → `feature_name` に変更:
        ```go
        case "feature_name":
            params.OldProgram = oldValue
            params.NewProgram = oldValue
        ```
        NewModule 構築:
        ```go
        if _, hasFeatureName := params.HintParams["feature_name"]; hasFeatureName {
            params.NewModule = "{{module_path}}/{{feature_name}}"
        } else {
            params.NewModule = "{{module_path}}"
        }
        ```
*   **Logic**:
    *   `Convert` 内で `ParamCollector` を生成
    *   Step 2 (AST transform): `newModule` に含まれるテンプレート変数を `pc.Add` で追加
    *   Step 4 (hints): `.hints` ファイルの `replace_with` フィールドに含まれるテンプレート変数を `pc.Add` で追加
    *   返り値として `pc` を返す

#### [MODIFY] [converter_test.go](file://features/templatizer/internal/converter/converter_test.go)
*   **Description**: `program_name` → `feature_name` 変更、`Convert` 返り値変更対応
*   **Logic**:
    *   全テストで `program_name` → `feature_name` に置換
    *   `{{module_path}}/{{program_name}}` → `{{module_path}}/{{feature_name}}` に置換
    *   `Convert` 呼び出しの返り値に `pc` (`*ParamCollector`) を追加
    *   `TestConvert` で `pc.Names()` に `"feature_name"`, `"module_path"` が含まれることを検証

#### [MODIFY] [ast_transformer_test.go](file://features/templatizer/internal/converter/ast_transformer_test.go)
*   **Description**: `{{program_name}}` → `{{feature_name}}` 変更
*   **Logic**: テストケース `"replaces long module path (real-world scaffold case)"` の `newModule` を `"{{module_path}}/{{feature_name}}"` に変更

#### [MODIFY] [hint_processor_test.go](file://features/templatizer/internal/converter/hint_processor_test.go)
*   **Description**: `program_name` → `feature_name` 変更
*   **Logic**: 全テストケースで `{{program_name}}` → `{{feature_name}}`, `"program_name"` → `"feature_name"` に置換

---

### main.go

#### [MODIFY] [main.go](file://features/templatizer/main.go)
*   **Description**: `processScaffold` で `ParamCollector` を取得し、`base_dir` のパラメータも蓄積、`generateShardFiles` でマージ
*   **Technical Design**:
    *   `processScaffold` の返り値に `*converter.ParamCollector` を追加:
        ```go
        func processScaffold(repoRoot string, s catalog.Scaffold, placement *catalog.Placement) (string, *converter.ParamCollector, error)
        ```
    *   `convertDefinitionsToScaffolds` はそのまま。`Placement` 情報を別途保持する `map[string]*catalog.Placement` を作成
*   **Logic**:
    1. `processScaffold` 内で `Convert` の返り値から `pc` を取得
    2. `placement.BaseDir` に含まれるテンプレート変数を `converter.ExtractTemplateVars` で抽出し `pc` に追加
    3. `pc` を返す
    4. `main()` で `processScaffold` の返り値から `pc` を受け取り、`converter.MergeParams(s.TemplateParams, pc.Names())` で最終的な `template_params` を構築
    5. `generateShardFiles` にマージ済み `template_params` を渡す

## Step-by-Step Implementation Guide

- [x] 1. **scaffold.yaml の `program_name` → `feature_name` 変更**
    - [x] `catalog/originals/axsh/go-standard-feature/scaffold.yaml` の `program_name` → `feature_name` に変更
    - [x] `catalog/originals/axsh/go-kotoshiro-mcp-feature/scaffold.yaml` の同上

- [x] 2. **`param_collector_test.go` 作成 (TDD: Red)**
    - [x] `TestExtractTemplateVars` 作成 (テーブル駆動、4ケース)
    - [x] `TestParamCollector` 作成 (Add, Names の検証)
    - [x] `TestMergeParams` 作成 (テーブル駆動、3ケース)

- [x] 3. **`param_collector.go` 実装 (TDD: Green)**
    - [x] `ParamCollector` 構造体と `NewParamCollector`, `Add`, `Names` メソッド
    - [x] `ExtractTemplateVars` 関数
    - [x] `MergeParams` 関数
    - [x] → テストPASS確認

- [x] 4. **`converter.go` 修正**
    - [x] `BuildConvertParams` で `program_name` → `feature_name` に変更
    - [x] `Convert` の返り値に `*ParamCollector` を追加
    - [x] `Convert` 内で `ParamCollector` を生成し各ステップでパラメータを蓄積

- [x] 5. **テスト更新 (`program_name` → `feature_name`)**
    - [x] `ast_transformer_test.go`: `{{program_name}}` → `{{feature_name}}`
    - [x] `hint_processor_test.go`: `program_name` → `feature_name`
    - [x] `converter_test.go`: `program_name` → `feature_name`、`Convert` 返り値対応、`ParamCollector` 検証追加

- [x] 6. **`main.go` 修正**
    - [x] `Placement` 情報を保持する map を追加
    - [x] `processScaffold` の返り値に `*converter.ParamCollector` 追加
    - [x] `base_dir` のテンプレート変数を `ParamCollector` に追加
    - [x] `MergeParams` で最終パラメータを構築
    - [x] `Scaffold.TemplateParams` をマージ結果で更新

- [x] 7. **ビルド & テスト検証**
    - [x] `./scripts/process/build.sh` 実行
    - [x] `./scripts/process/integration_test.sh` 実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `TestExtractTemplateVars` 全ケースPASS
    *   `TestParamCollector` PASS
    *   `TestMergeParams` 全ケースPASS
    *   `TestTransformGoMod` 7ケースPASS（`feature_name` 使用）
    *   `TestConvert` 3ケースPASS + `ParamCollector` 検証
    *   `TestBuildConvertParamsOldValueFallback` 3ケースPASS
    *   `TestProcessHints` 5ケースPASS

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   リグレッションなし

## Documentation

本変更は内部実装の修正であり、既存ドキュメントへの影響はありません。
