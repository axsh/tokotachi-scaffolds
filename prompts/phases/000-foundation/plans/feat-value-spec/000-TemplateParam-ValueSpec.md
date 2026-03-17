# 000-TemplateParam-ValueSpec

> **Source Specification**: [000-TemplateParam-ValueSpec.md](file://prompts/phases/000-foundation/ideas/feat-value-spec/000-TemplateParam-ValueSpec.md)

## Goal Description

`template_params` に `value_spec` フィールドを追加し、各パラメータに型（type）、長さ制約（length）、フォーマット制約（format）、範囲制約（range）、列挙制約（enum）を定義可能にする。`templatizer` が未定義パラメータを自動追加する際に、デフォルトの `ValueSpec`（`type: string`, `max_bytes: 256`）を付与する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 型定義（type: string/number） | Proposed Changes > catalog.go: `ValueSpec.Type` |
| 長さ制約（max_bytes, max_chars, max_digits） | Proposed Changes > catalog.go: `LengthSpec` |
| フォーマット制約（pattern） | Proposed Changes > catalog.go: `FormatSpec` |
| 範囲制約（minimum/maximum/exclusive_*） | Proposed Changes > catalog.go: `RangeSpec` |
| enum制約 | Proposed Changes > catalog.go: `ValueSpec.Enum` |
| デフォルト ValueSpec 付与（auto-add時） | Proposed Changes > param_collector.go: `MergeParams` |
| 既存 value_spec の尊重 | Proposed Changes > param_collector.go: `MergeParams` |
| scaffold.yaml への value_spec 記載 | Proposed Changes > scaffold.yaml（2ファイル） |
| ドキュメント更新 | Documentation > 000-Reference-Manual.md |

## Proposed Changes

### catalog パッケージ（データモデル）

#### [MODIFY] [catalog_test.go](file://features/templatizer/internal/catalog/catalog_test.go)

*   **Description**: `ValueSpec` のパース・シリアライズに関する単体テストを追加
*   **Technical Design**:
    *   既存の `TestParseCatalogWithTemplateParams` にテストケースを追加
    *   新規テスト関数 `TestParseCatalogWithValueSpec` を追加
*   **Logic**:
    *   テストケース "scaffold with value_spec (string type)":
        *   YAML入力に `value_spec` を含む `template_params` を定義
        *   `value_spec.type` = "string", `value_spec.length.max_bytes` = 256, `value_spec.format.pattern` = "^[a-z]+$" を検証
    *   テストケース "scaffold with value_spec (number type with range)":
        *   `value_spec.type` = "number", `value_spec.range.minimum` = 1, `value_spec.range.maximum` = 65535 を検証
    *   テストケース "scaffold with value_spec (enum)":
        *   `value_spec.enum` = ["debug", "release"] を検証
    *   テストケース "scaffold without value_spec":
        *   `ValueSpec` が `nil` であることを検証
    *   テストケース "scaffold definition with value_spec":
        *   `ParseScaffoldDefinition` 経由でも `ValueSpec` が正しくパースされることを検証

---

#### [MODIFY] [catalog.go](file://features/templatizer/internal/catalog/catalog.go)

*   **Description**: `ValueSpec` 関連構造体の追加と `TemplateParam` への `ValueSpec` フィールド追加
*   **Technical Design**:
    *   4つの新規構造体を追加: `ValueSpec`, `LengthSpec`, `FormatSpec`, `RangeSpec`
    *   `TemplateParam` に `ValueSpec *ValueSpec` フィールドを追加
    *   デフォルト値生成ヘルパー関数 `DefaultValueSpec()` を追加
    ```go
    // ValueSpec defines validation rules for a template parameter value.
    type ValueSpec struct {
        Type   string      `yaml:"type,omitempty"`   // "string" or "number"
        Length *LengthSpec `yaml:"length,omitempty"`
        Format *FormatSpec `yaml:"format,omitempty"`
        Range  *RangeSpec  `yaml:"range,omitempty"`
        Enum   []string    `yaml:"enum,omitempty"`
    }

    // LengthSpec defines length constraints for parameter values.
    type LengthSpec struct {
        MaxBytes  *int `yaml:"max_bytes,omitempty"`
        MaxChars  *int `yaml:"max_chars,omitempty"`
        MaxDigits *int `yaml:"max_digits,omitempty"`
    }

    // FormatSpec defines format constraints using regular expressions.
    type FormatSpec struct {
        Pattern string `yaml:"pattern,omitempty"`
    }

    // RangeSpec defines numeric range constraints (JSONSchema style).
    type RangeSpec struct {
        Minimum          *float64 `yaml:"minimum,omitempty"`
        Maximum          *float64 `yaml:"maximum,omitempty"`
        ExclusiveMinimum *float64 `yaml:"exclusive_minimum,omitempty"`
        ExclusiveMaximum *float64 `yaml:"exclusive_maximum,omitempty"`
    }
    ```
    *   `TemplateParam` の変更:
    ```go
    type TemplateParam struct {
        Name        string     `yaml:"name"`
        Description string     `yaml:"description"`
        Required    bool       `yaml:"required"`
        Default     string     `yaml:"default,omitempty"`
        OldValue    string     `yaml:"old_value"`
        ValueSpec   *ValueSpec `yaml:"value_spec,omitempty"`  // NEW
    }
    ```
    *   `DefaultValueSpec()` 関数:
    ```go
    // DefaultValueSpec returns the default ValueSpec for auto-added parameters.
    // Type: "string", MaxBytes: 256.
    func DefaultValueSpec() *ValueSpec {
        maxBytes := 256
        return &ValueSpec{
            Type: "string",
            Length: &LengthSpec{
                MaxBytes: &maxBytes,
            },
        }
    }
    ```

---

### converter パッケージ（パラメータマージ）

#### [MODIFY] [param_collector_test.go](file://features/templatizer/internal/converter/param_collector_test.go)

*   **Description**: `MergeParams` のデフォルト `ValueSpec` 付与テストを追加
*   **Technical Design**:
    *   既存の `TestMergeParams` にサブテストを追加
*   **Logic**:
    *   テストケース "auto-added param has default ValueSpec":
        *   defined: `module_path`（ValueSpec付き）のみ
        *   discovered: `["feature_name", "module_path"]`
        *   `MergeParams` 実行後:
            *   `feature_name` に `ValueSpec` が付与されている（`Type == "string"`, `Length.MaxBytes == 256`）
            *   `module_path` の元の `ValueSpec` が変更されていない
    *   テストケース "preserves existing ValueSpec from defined":
        *   defined: `module_path`（カスタム ValueSpec: `type: string`, `max_bytes: 512`, `pattern: "^[a-zA-Z0-9._/-]+$"`）
        *   discovered: `["module_path"]`
        *   `MergeParams` 実行後:
            *   `module_path.ValueSpec.Length.MaxBytes` == 512（上書きされない）
            *   `module_path.ValueSpec.Format.Pattern` == "^[a-zA-Z0-9._/-]+$"（保持）

---

#### [MODIFY] [param_collector.go](file://features/templatizer/internal/converter/param_collector.go)

*   **Description**: `MergeParams` の auto-add ロジックにデフォルト `ValueSpec` を付与
*   **Technical Design**:
    *   `MergeParams` 関数内、未定義パラメータの auto-add 箇所を修正
    *   現在の auto-add コード（L97-100）:
    ```go
    merged[name] = catalog.TemplateParam{
        Name:     name,
        Required: true,
    }
    ```
    *   修正後:
    ```go
    merged[name] = catalog.TemplateParam{
        Name:      name,
        Required:  true,
        ValueSpec: catalog.DefaultValueSpec(),
    }
    ```

---

### scaffold.yaml（既存テンプレート定義）

#### [MODIFY] [scaffold.yaml](file://catalog/originals/axsh/go-kotoshiro-mcp-feature/scaffold.yaml)

*   **Description**: `template_params` に `value_spec` を追加
*   **Logic**:
    *   `module_path`: type=string, max_bytes=256, pattern=`^[a-zA-Z0-9._/-]+$`
    *   `feature_name`: type=string, max_bytes=64, max_chars=32, pattern=`^[a-z][a-z0-9_-]*$`
    ```yaml
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        default: "github.com/axsh/tokotachi/features/myfunction"
        value_spec:
          type: string
          length:
            max_bytes: 256
          format:
            pattern: "^[a-zA-Z0-9._/-]+$"
      - name: "feature_name"
        description: "Feature name"
        required: true
        default: "myfunction"
        value_spec:
          type: string
          length:
            max_bytes: 64
            max_chars: 32
          format:
            pattern: "^[a-z][a-z0-9_-]*$"
    ```

---

#### [MODIFY] [scaffold.yaml](file://catalog/originals/axsh/go-standard-feature/scaffold.yaml)

*   **Description**: `template_params` に `value_spec` を追加（go-kotoshiro-mcp-feature と同じ構造）
*   **Logic**:
    *   `module_path`: type=string, max_bytes=256, pattern=`^[a-zA-Z0-9._/-]+$`
    *   `feature_name`: type=string, max_bytes=64, max_chars=32, pattern=`^[a-z][a-z0-9_-]*$`
    *   YAML構造は go-kotoshiro-mcp-feature と同一

---

## Step-by-Step Implementation Guide

### Phase 1: テスト先行（TDD）

- [x] 1. **catalog_test.go にValueSpecパーステストを追加**
    *   `TestParseCatalogWithValueSpec` テスト関数を追加
    *   テストケース: string型, number型+range, enum, ValueSpec未指定, ScaffoldDefinitionパース
    *   この時点でテストは失敗する（`ValueSpec` 構造体が未定義）

- [x] 2. **param_collector_test.go にデフォルトValueSpecテストを追加**
    *   `TestMergeParams` に "auto-added param has default ValueSpec" サブテストを追加
    *   `TestMergeParams` に "preserves existing ValueSpec from defined" サブテストを追加
    *   この時点でテストは失敗する

### Phase 2: データモデル実装

- [x] 3. **catalog.go にValueSpec構造体を追加**
    *   `ValueSpec`, `LengthSpec`, `FormatSpec`, `RangeSpec` 構造体を定義
    *   `TemplateParam` に `ValueSpec *ValueSpec` フィールドを追加
    *   `DefaultValueSpec()` ヘルパー関数を追加
    *   ビルド確認:
    ```bash
    ./scripts/process/build.sh
    ```

### Phase 3: MergeParamsロジック更新

- [x] 4. **param_collector.go の MergeParams を修正**
    *   auto-add 箇所に `ValueSpec: catalog.DefaultValueSpec()` を追加
    *   ビルド確認:
    ```bash
    ./scripts/process/build.sh
    ```

### Phase 4: scaffold.yaml 更新

- [x] 5. **go-kotoshiro-mcp-feature/scaffold.yaml に value_spec を追加**
    *   `module_path`, `feature_name` の各パラメータに `value_spec` を記載

- [x] 6. **go-standard-feature/scaffold.yaml に value_spec を追加**
    *   同様の `value_spec` を記載

### Phase 5: templatizer 実行と統合テスト

- [x] 7. **templatizer を実行して出力を確認**
    *   ビルドと統合テストを実行:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   生成されたシャードファイル（例: `catalog/scaffolds/l/r/y/6.yaml`）に `value_spec` が含まれていることを確認

### Phase 6: ドキュメント更新

- [x] 8. **000-Reference-Manual.md を更新**
    *   `template_params` セクションに `value_spec` の仕様を追記

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    全ユニットテスト（ValueSpec パース、MergeParams デフォルト付与）がパスすることを確認。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    統合テストを実行し、templatizer の出力に value_spec が反映されていることを確認。
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: シャードYAMLファイル内に `value_spec` セクションが出力されていること。auto-add されたパラメータに `type: string`, `max_bytes: 256` が含まれていること。

## Documentation

#### [MODIFY] [000-Reference-Manual.md](file://prompts/specifications/000-Reference-Manual.md)
*   **更新内容**: 「テンプレートパラメータ (`template_params`)」セクションに `value_spec` の仕様を追記
    *   `value_spec` フィールドの説明
    *   `type`, `length`, `format`, `range`, `enum` の各サブフィールドの説明テーブル
    *   YAML記述例
    *   デフォルト挙動（templatizer auto-add 時: `type: string`, `max_bytes: 256`）
