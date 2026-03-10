# 002-Templatizer-ParamUnification

> **Source Specification**: [002-Templatizer-ParamUnification.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/002-Templatizer-ParamUnification.md)

## Goal Description

`catalog.yaml` の `options` フィールドを廃止し、`template_params` に一本化する。`old_value` 省略時は `default` にフォールバックする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: options 廃止と template_params への統合 | Proposed Changes > catalog.yaml |
| R2: catalog.yaml の更新 | Proposed Changes > catalog.yaml |
| R3: Scaffold 構造体の Options 削除 | 不要（現状 Options フィールドは存在しない） |
| R4: BuildConvertParams の old_value フォールバック | Proposed Changes > converter.go |

## Proposed Changes

### converter パッケージ

#### [NEW] BuildConvertParams のフォールバックテスト追加 [converter_test.go](file://features/templatizer/internal/converter/converter_test.go)
*   **Description**: `old_value` 省略時に `default` がフォールバックされるテストを追加（TDD）
*   **Logic (テストケース追加)**:
    1. `OldValue: ""`, `Default: "github.com/axsh/tokotachi/features/myprog"` → `OldModule` が `default` の値になること
    2. `OldValue: "function"`, `Default: "something-else"` → `OldModule` が `OldValue` の `"function"` のままであること（明示指定優先）

#### [MODIFY] [converter.go](file://features/templatizer/internal/converter/converter.go)
*   **Description**: `BuildConvertParams` に `old_value` → `default` フォールバックを追加
*   **Technical Design**:
    ```go
    // BuildConvertParams 内のループで old_value を解決
    for _, tp := range templateParams {
        // Resolve old_value: explicit old_value takes priority, fallback to default.
        oldValue := tp.OldValue
        if oldValue == "" {
            oldValue = tp.Default
        }

        switch tp.Name {
        case "module_path":
            params.OldModule = oldValue
            params.NewModule = oldValue
        case "program_name":
            params.OldProgram = oldValue
            params.NewProgram = oldValue
        }
        params.HintParams[tp.Name] = oldValue
    }
    ```

---

### catalog.yaml

#### [MODIFY] [catalog.yaml](file://catalog.yaml)
*   **Description**: `options` フィールドを削除し、`template_params` に `default` を追加
*   **Logic**:
    - `axsh-go-standard` feature:
      - `options` ブロック全体を削除
      - `template_params` の各エントリに `default` を追加（`old_value` は削除、`default` と同値のため）
    - `axsh-go-kotoshiro-mcp` feature:
      - `options` ブロック全体を削除
      - `template_params` の各エントリに `default` を追加（`old_value` は削除）

**変更後の `axsh-go-standard` feature:**
```yaml
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myprog"
  - name: "program_name"
    description: "Program name"
    required: true
    default: "myprog"
```

**変更後の `axsh-go-kotoshiro-mcp` feature:**
```yaml
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "function"
  - name: "program_name"
    description: "Program name"
    required: true
    default: "function"
```

## Step-by-Step Implementation Guide

1. **[x] BuildConvertParams フォールバックテスト追加**:
    *   `converter_test.go` に `TestBuildConvertParamsOldValueFallback` テストを追加
    *   テストが FAIL することを確認（TDD）

2. **[x] BuildConvertParams フォールバック実装**:
    *   `converter.go` の `BuildConvertParams` で `oldValue` フォールバックロジックを追加
    *   テストが PASS することを確認

3. **[x] catalog.yaml 更新**:
    *   `options` ブロックを削除
    *   `template_params` に `default` を追加、`old_value` を削除

4. **[x] ビルドパイプライン実行**:
    *   `scripts/process/build.sh` で全テスト通過を確認

## Verification Plan

### Automated Verification

```bash
./scripts/process/build.sh
```
* 全ての既存テスト + 新規フォールバックテストが PASS すること

## Documentation

#### [MODIFY] [001-Templatizer-TemplateConversion.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/001-Templatizer-TemplateConversion.md)
*   **更新内容**: `catalog.yaml の拡張` セクション（L333-412）で `options` への言及を削除し、`template_params` の `old_value` 省略時フォールバックについて追記
