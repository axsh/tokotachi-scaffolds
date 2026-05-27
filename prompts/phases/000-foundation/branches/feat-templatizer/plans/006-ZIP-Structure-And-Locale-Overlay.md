# 006-ZIP-Structure-And-Locale-Overlay

> **Source Specification**: [006-ZIP-Structure-And-Locale-Overlay.md](file://prompts/phases/000-foundation/ideas/feat-templatizer/006-ZIP-Structure-And-Locale-Overlay.md)

## Goal Description

`001-HowToExtract.md` と `000-Reference-Manual.md` の ZIP 展開例を実際の構造（`base/` + `locale.<lang>/` + `scaffold.yaml`）に修正し、ロケールオーバーレイ処理の手順を明確化する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: ZIP 展開例の修正 | Proposed Changes > 001-HowToExtract.md, 000-Reference-Manual.md |
| R2: ロケールオーバーレイ処理の明確化 | Proposed Changes > 001-HowToExtract.md (Step 3.2, 3.3) |
| R3: scaffold.yaml の除外ルール | Proposed Changes > 001-HowToExtract.md (Step 3.2) |

## Proposed Changes

### ドキュメント

#### [MODIFY] [001-HowToExtract.md](file://prompts/specifications/001-HowToExtract.md)

*   **Description**: ZIP 展開例を実態に合わせ、ロケールオーバーレイの処理手順を具体化
*   **Logic**:
    *   Step 3.2「ZIP の展開」の構造例を以下に修正:
        ```
        # ZIP 展開後の構造例（root/project-default の場合）
        base/
          AGENTS.md
          features/
            README.md
          prompts/
            ...
          scripts/
            .gitkeep
          shared/
            README.md
        locale.ja/
          features/
            README.md
          prompts/
            phases/
              README.md
          shared/
            README.md
        scaffold.yaml
        ```
    *   Step 3.2 に「ZIP 内の `scaffold.yaml` はクライアント側の展開対象から除外する」注記を追加
    *   Step 3.3「ロケールオーバーレイの適用」を以下の具体的な手順に修正:
        1. ZIP を展開（テンポラリディレクトリ）
        2. ユーザーのロケールを検出（`LANG`, `LC_ALL`, `--lang`）
        3. 該当 `locale.<lang>/` が ZIP 内に存在するか確認
        4. **存在する場合**: `base/` をコピー → `locale.<lang>/` で上書き（同名ファイルのみ置換）
        5. **存在しない場合**: `base/` のみをコピー
        6. 結果として得られたファイル群を以降のテンプレート処理に使用

#### [MODIFY] [000-Reference-Manual.md](file://prompts/specifications/000-Reference-Manual.md)

*   **Description**: ZIP 構造例を `base/` + `locale.<lang>/` 構造に修正
*   **Logic**:
    *   templatizer 処理フロー付近の ZIP 構造例を修正
    *   `scaffold.yaml` が ZIP 内に含まれるがクライアントは無視する旨を追記

## Step-by-Step Implementation Guide

- [x] **Step 1: 001-HowToExtract.md の修正**
    - Step 3.2 の ZIP 展開例を `base/` + `locale.<lang>/` + `scaffold.yaml` 構造に修正
    - `scaffold.yaml` 除外ルールの注記を追加
    - Step 3.3 のロケールオーバーレイを具体的な手順に修正

- [x] **Step 2: 000-Reference-Manual.md の修正**
    - ZIP 構造例を `base/` + `locale.<lang>/` 構造に修正
    - `scaffold.yaml` の除外ルールを追記

## Verification Plan

### Automated Verification

ドキュメント変更のみのため、ビルドパイプラインへの影響なし。

1.  **Build & Unit Tests** (リグレッション確認):
    ```bash
    ./scripts/process/build.sh
    ```

## Documentation

本計画自体がドキュメント更新であるため、上記 Proposed Changes が対象。
