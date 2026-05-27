# catalog.yaml パラメータ統合: options と template_params の一本化

## 背景 (Background)

前仕様 (`001-Templatizer-TemplateConversion.md`) において、テンプレート変換パイプラインを実装した。
その中で `catalog.yaml` に以下の **2つのパラメータ定義** が存在する状態となった：

```yaml
# 現状の catalog.yaml（抜粋）
scaffolds:
  - name: "axsh-go-standard"
    options:                          # scaffold 展開時のユーザー入力パラメータ
      - name: "feature_name"
        default: "myprog"
      - name: "go_module"
        default: "github.com/axsh/tokotachi/features"
    template_params:                  # templatizer のコード変換パラメータ
      - name: "module_path"
        old_value: "github.com/axsh/tokotachi/features/myprog"
      - name: "program_name"
        old_value: "myprog"
```

### 現在の課題

1. **意味的な重複**: `options.feature_name` と `template_params.program_name` は実質同じ値（`myprog`）を指し、`options.go_module` + `options.feature_name` を結合すると `template_params.module_path` と一致する
2. **管理の二重化**: パラメータを追加・変更するたびに `options` と `template_params` の両方を更新する必要がある
3. **`old_value` と `default` の冗長性**: `old_value`（originals 内の置換元文字列）と `default`（ユーザー未入力時の既定値）が同じ値になるケースが多い

## 要件 (Requirements)

### 必須要件

#### R1: `options` の廃止と `template_params` への統合

`options` フィールドを廃止し、全てのパラメータ定義を `template_params` に一本化する。

**統合後の `template_params` の構造:**

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `name` | string | ✔ | パラメータ名（`{{name}}` として hints 内で参照） |
| `description` | string | ✔ | パラメータの説明 |
| `required` | bool | ✔ | ユーザーに入力を必須とするか |
| `default` | string | △ | ユーザー未入力時の既定値（`required: false` の場合に使用） |
| `old_value` | string | △ | originals 内の置換元文字列。**省略時は `default` の値が使用される** |

> [!IMPORTANT]
> `old_value` 省略時のフォールバック:
> - `old_value` が明示的に指定されている → その値を置換元として使用
> - `old_value` が省略されている → `default` の値を置換元として使用
> - 両方省略されている → そのパラメータは変換に使用しない（hints 専用パラメータの可能性）

#### R2: `catalog.yaml` の更新

既存の scaffold エントリから `options` を削除し、必要な情報を `template_params` に統合する。

**変換前:**
```yaml
options:
  - name: "feature_name"
    description: "Feature name"
    required: true
    default: "myprog"
  - name: "go_module"
    description: "Go module base name"
    required: false
    default: "github.com/axsh/tokotachi/features"
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    old_value: "github.com/axsh/tokotachi/features/myprog"
  - name: "program_name"
    description: "Program name"
    required: true
    old_value: "myprog"
```

**変換後:**
```yaml
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myprog"
    # old_value 省略 → default が置換元
  - name: "program_name"
    description: "Program name"
    required: true
    default: "myprog"
    # old_value 省略 → default が置換元
```

> `options` にあった `feature_name` と `go_module` は `template_params` の `program_name` と `module_path` に統合される。2パラメータ→2パラメータで変化なし。

#### R3: `Scaffold` 構造体から `Options` フィールドを削除

Go コード上の `catalog.Scaffold` 構造体で `Options` フィールドが存在する場合は削除する。

> 現状、`Options` フィールドは `Scaffold` 構造体に定義されておらず、YAML パース時に無視されている。構造体の変更は不要だが、`catalog.yaml` 上の `options` エントリは削除する必要がある。

#### R4: `BuildConvertParams` の `old_value` フォールバック対応

`converter.BuildConvertParams` 関数で、`TemplateParam.OldValue` が空文字列の場合に `TemplateParam.Default` にフォールバックするロジックを追加する。

```go
// BuildConvertParams 内の変更イメージ:
oldValue := tp.OldValue
if oldValue == "" {
    oldValue = tp.Default
}
```

## 実現方針 (Implementation Approach)

### 変更対象

```
catalog.yaml                                        # options 削除、template_params 統合
features/templatizer/internal/catalog/catalog.go     # TemplateParam に Default フィールド確認
features/templatizer/internal/converter/converter.go # BuildConvertParams の old_value フォールバック
```

### 影響範囲

- **既存テスト**: `catalog_test.go` の `TestParseCatalogWithTemplateParams` は `default` フィールドのパースを既にサポートしている（`Default string \`yaml:"default,omitempty"\``）ため、構造体変更は不要
- **`BuildConvertParams`**: フォールバックロジックの追加のみ
- **`catalog.yaml`**: YAML 内容の編集（フォーマット変更）

## 検証シナリオ (Verification Scenarios)

### シナリオ1: catalog.yaml パース確認

1. `catalog.yaml` から `options` を削除し、`template_params` に `default` を追加する
2. templatizer を実行し、正常に scaffold が処理されることを確認する
3. `old_value` が省略されたパラメータで `default` が置換元として使用されることを確認する

### シナリオ2: old_value フォールバック

1. `template_params` に `old_value` を省略し `default` のみ指定したエントリを用意する
2. `BuildConvertParams` で `default` の値が `OldModule` / `OldProgram` に設定されることを確認する

## テスト項目 (Testing for the Requirements)

### 単体テスト

| テスト対象 | テスト内容 | テストファイル |
|---|---|---|
| BuildConvertParams | `old_value` 未指定時に `default` がフォールバックされること | `internal/converter/converter_test.go` |
| BuildConvertParams | `old_value` 指定時はそちらが優先されること | `internal/converter/converter_test.go` |

### 検証コマンド

```bash
./scripts/process/build.sh
```
