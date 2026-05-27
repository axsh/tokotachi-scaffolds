# 010-GoMod-ModulePath-FeatureName-Split

> **Source Specification**: [010-GoMod-ModulePath-FeatureName-Split.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/010-GoMod-ModulePath-FeatureName-Split.md)

## Goal Description

`BuildConvertParams` 関数で `module_path` の resolved value（default または old_value）の末尾に `feature_name` が含まれている場合、それを除外して `module_path` のHintParams値を正しく設定する。これにより、生成されるscaffold.yamlの `template_params` で `module_path` と `feature_name` の重複が解消される。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| go.mod module行のパース改善（末尾feature_name除外） | Proposed Changes > converter.go `BuildConvertParams` |
| scaffold.yaml の template_params 出力修正 | Proposed Changes > scaffold.yaml default値修正 |
| 既存テストの更新 | Proposed Changes > converter_test.go テストケース更新・追加 |
| feature_name不一致時のフォールバック | Proposed Changes > converter.go + converter_test.go |
| feature_name未定義時の現行動作維持 | Proposed Changes > converter_test.go 既存ケース維持 |

## Proposed Changes

### converter パッケージ

#### [MODIFY] [converter_test.go](file://features/templatizer/internal/converter/converter_test.go)

*   **Description**: `TestBuildConvertParamsOldValueFallback` の既存テストケースの期待値を修正し、新しいテストケースを追加する。
*   **Technical Design**:
    *   テストケース `"falls back to default when old_value is empty"` の期待値更新:
        *   `HintParams["module_path"]`: `"github.com/axsh/tokotachi/features/myprog"` → `"github.com/axsh/tokotachi/features"` に変更
        *   `OldModule` は変更なし（フルパス `"github.com/axsh/tokotachi/features/myprog"` のまま。OldModule は go.mod の module行置換に使うため分離しない）
    *   テストケース `"explicit old_value takes priority over default"` の期待値確認:
        *   `module_path` OldValue=`"function"`, `feature_name` OldValue=`"function"` の場合: `module_path` は `"function"` のまま末尾が `feature_name` と一致するが、一致した場合は末尾を除外 → 空文字列になるのは不正なので、1セグメントのみの場合は分離しない
        *   → OldModule = `"function"`, HintParams[module_path] = `"function"` のまま（末尾除外で空になるケースはフォールバック）
    *   新規テストケース `"module_path suffix matches feature_name — suffix stripped for HintParams"`:
        *   入力: `module_path` default=`"github.com/org/features/myapp"`, `feature_name` default=`"myapp"`
        *   期待: `OldModule` = `"github.com/org/features/myapp"`, `HintParams["module_path"]` = `"github.com/org/features"`
    *   新規テストケース `"module_path suffix does not match feature_name — no stripping"`:
        *   入力: `module_path` default=`"github.com/org/app"`, `feature_name` default=`"other"`
        *   期待: `OldModule` = `"github.com/org/app"`, `HintParams["module_path"]` = `"github.com/org/app"`
*   **Logic**:
    *   各テストケースで `BuildConvertParams` を呼び出し、`OldModule`, `HintParams["module_path"]`, `OldProgram`, `NewModule` を検証

---

#### [MODIFY] [converter.go](file://features/templatizer/internal/converter/converter.go)

*   **Description**: `BuildConvertParams` 関数に module_path/feature_name 分離ロジックを追加する。
*   **Technical Design**:
    *   既存の `for _, tp := range templateParams` ループは変更なし
    *   ループ後、`HintParams["module_path"]` と `HintParams["feature_name"]` の両方が存在する場合に分離ロジックを実行
    *   新規ヘルパー関数 `stripFeatureNameSuffix(modulePath, featureName string) string` を追加
*   **Logic**:
    *   `stripFeatureNameSuffix` の処理:
        1. `modulePath` が `/<featureName>` で終わるかチェック（`strings.HasSuffix(modulePath, "/"+featureName)`）
        2. 一致する場合: `modulePath[:len(modulePath)-len(featureName)-1]` を返す（末尾の `/<featureName>` を除去）
        3. 一致しない場合、または結果が空文字列になる場合: `modulePath` をそのまま返す（フォールバック）
    *   `BuildConvertParams` の既存テンプレート変数構築セクション（L109-L116）の**前**に、以下を追加:
        ```go
        // Strip feature_name suffix from module_path HintParams value.
        if featureName, ok := params.HintParams["feature_name"]; ok {
            if mp, ok := params.HintParams["module_path"]; ok {
                params.HintParams["module_path"] = stripFeatureNameSuffix(mp, featureName)
            }
        }
        ```
    *   `OldModule` は変更しない。`OldModule` は TransformGoMod / TransformGoSource で元のgo.modのmodule行を検索置換するために使う値なので、フルパス（`feature_name` 含む）のままである必要がある

---

### scaffold.yaml 定義ファイル

#### [MODIFY] [scaffold.yaml](file://catalog/originals/axsh/go-standard-feature/scaffold.yaml)

*   **Description**: `module_path` の `default` 値を修正する。
*   **Logic**:
    *   L20 の `default: "github.com/axsh/tokotachi/features/myprog"` を `default: "github.com/axsh/tokotachi/features"` に変更

## Step-by-Step Implementation Guide

1. [x] **テスト更新（TDD: Red）**:
    *   `features/templatizer/internal/converter/converter_test.go` を編集
    *   `"falls back to default when old_value is empty"` テストの `HintParams["module_path"]` 期待値を `"github.com/axsh/tokotachi/features"` に変更
    *   新規テストケース `"module_path suffix matches feature_name"` を追加
    *   新規テストケース `"module_path suffix does not match feature_name"` を追加
    *   ビルドし、テストが **FAIL** することを確認

2. [x] **実装（TDD: Green）**:
    *   `features/templatizer/internal/converter/converter.go` を編集
    *   `stripFeatureNameSuffix(modulePath, featureName string) string` 関数を追加
    *   `BuildConvertParams` のテンプレート変数構築セクションの前に分離ロジックを追加
    *   ビルドし、テストが **PASS** することを確認

3. [x] **scaffold.yaml 修正**:
    *   `catalog/originals/axsh/go-standard-feature/scaffold.yaml` の `module_path` default を修正

4. [x] **全体ビルド検証**:
    *   `./scripts/process/build.sh` を実行し、全テスト PASS を確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**: `TestBuildConvertParamsOldValueFallback` の全サブテストが PASS すること
    *   **確認事項**: `TestConvert` の全サブテストが PASS すること（既存のパイプラインテストでリグレッションなし）

## Documentation

既存の仕様書・ドキュメントへの更新は不要です。
