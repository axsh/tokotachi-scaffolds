# ZIP 展開構造とロケールオーバーレイの明確化

## 背景 (Background)

`001-HowToExtract.md` の ZIP 展開例が、実際の ZIP 内構造と一致していない。

**ドキュメントの記載（誤）:**

```
# ZIP 展開後の構造例
go.mod.tmpl
cmd/myfunction/
  main.go.tmpl
internal/
  handler/
    handler.go
README.md
```

**実際の ZIP 内構造（正）:**

```
base/AGENTS.md
base/features/README.md
base/prompts/phases/...
base/scripts/.gitkeep
base/shared/...
base/work/README.md
locale.ja/features/README.md
locale.ja/prompts/phases/README.md
locale.ja/shared/...
locale.ja/work/README.md
scaffold.yaml
```

- ZIP は `originals/{org}/{project}/` ディレクトリ全体をアーカイブしている
- `base/` と `locale.<lang>/` がトップレベルに含まれる
- `scaffold.yaml`（scaffold 定義）も ZIP 内に含まれる
- ロケールオーバーレイの説明は存在するが、ZIP 構造と整合しておらず、具体的な適用手順が不明瞭

## 要件 (Requirements)

### 必須要件

#### R1: ドキュメントの ZIP 展開例の修正

`001-HowToExtract.md` と `000-Reference-Manual.md` の ZIP 展開例を、実際の構造（`base/` + `locale.<lang>/` + `scaffold.yaml`）に合わせて修正する。

#### R2: ロケールオーバーレイ処理の明確化

クライアント（`tt scaffold`）がロケールを適用する具体的な手順を文書化する。

**処理フロー:**

1. ZIP を展開する
2. ユーザーのロケールを検出する（`LANG`, `LC_ALL`, `--lang` フラグ）
3. 該当する `locale.<lang>/` ディレクトリが ZIP 内に存在するか確認する
4. 存在する場合:
   - `base/` のファイルを作業ディレクトリにコピー
   - `locale.<lang>/` のファイルで上書き（マージ）
5. 存在しない場合:
   - `base/` のファイルのみを作業ディレクトリにコピー
6. `scaffold.yaml` はテンプレートファイルではないため、展開対象から除外

#### R3: scaffold.yaml の除外ルール

ZIP 内の `scaffold.yaml` はシャーディング YAML と同内容であり、クライアント側の展開対象に含めない。クライアントはシャーディング YAML から scaffold 情報を取得済みのため、ZIP 内の `scaffold.yaml` は無視する。

## 実現方針 (Implementation Approach)

### 変更対象

```
prompts/specifications/001-HowToExtract.md    # ZIP 展開例とロケールオーバーレイの修正
prompts/specifications/000-Reference-Manual.md # ZIP 構造例の修正（該当箇所のみ）
```

### 変更内容

- ZIP 展開後の構造例を `base/` + `locale.<lang>/` を含む形式に修正
- ロケールオーバーレイの処理手順を具体的な順序で記載
- `scaffold.yaml` の除外ルールを追記

## 検証シナリオ (Verification Scenarios)

### シナリオ1: ZIP 展開例の正確性

1. `project-default.zip` を展開する
2. トップレベルに `base/`, `locale.ja/`, `scaffold.yaml` が存在することを確認
3. ドキュメントの構造例がこの実態と一致していることを確認

### シナリオ2: ロケールオーバーレイの手順

1. `base/features/README.md` と `locale.ja/features/README.md` が両方存在する
2. ロケール `ja` の場合、`locale.ja/features/README.md` が優先される
3. `locale.ja/` に存在しないファイル（例: `base/AGENTS.md`）は `base/` から使用される

## テスト項目 (Testing for the Requirements)

### ドキュメントレビュー

ドキュメント変更のみのため、自動テスト対象外。手動でドキュメントの正確性を確認する。

```bash
# ZIP の実際の構造を確認
unzip -l catalog/scaffolds/6/j/v/project-default.zip
```
