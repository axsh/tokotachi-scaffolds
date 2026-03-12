# 008: go.mod テンプレート化の修正

## 背景 (Background)

`templatizer` がスキャフォルドZIPを生成する際、`go.mod` ファイルの `module` 行がテンプレート変数に変換されていない。

### 現状の問題

`catalog/originals/axsh/go-standard-feature/base/go.mod` の内容:

```
module github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature

go 1.24.0
```

生成されたZIP (`catalog/scaffolds/b/i/b/go-standard-feature.zip`) を展開すると、`go.mod` がそのまま残っており、以下のようにテンプレート化されていない:

```
module github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature
```

**期待される結果:**

```
module {{module_path}}/{{program_name}}
```

### 根本原因

`TransformGoMod` 関数 (`ast_transformer.go`) は、`oldModule` 引数と `go.mod` の `module` 行の**完全一致 (exact match)** でしか変換を行わない。

```go
currentModule := strings.TrimSpace(match[1])
if currentModule != oldModule {
    return content, false, nil  // ← 不一致のためスキップ
}
```

`BuildConvertParams` (`converter.go`) は `OldModule` と `NewModule` を `scaffold.yaml` の `template_params` から構築するが、`OldModule` には `old_value` → `default` のフォールバック値が使用される:

- `OldModule` = `github.com/axsh/tokotachi/features/myprog` (scaffold.yaml の default)
- 実際の `go.mod` の module = `github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature`

この2つが一致しないため、`TransformGoMod` は変換をスキップし、`go.mod` はテンプレート化されない。

## 要件 (Requirements)

### 必須要件

1. **go.mod の module 行を自動的にテンプレート化する**: `TransformGoMod` が `oldModule` 引数と一致しない場合でも、`go.mod` の `module` 行を読み取り、テンプレート変数 `{{module_path}}/{{program_name}}` に置換してテンプレート化 (`.tmpl` にリネーム) を行うこと。

2. **テンプレート化のロジック変更**: `TransformGoMod` の引数に `oldModule` を渡してexact matchする設計から、`go.mod` のmodule行を直接読み取り`newModule`（テンプレート変数）に置き換える設計に変更する。

3. **newModule の構成**: `scaffold.yaml` の `template_params` にある `module_path` と `program_name` を使い、`{{module_path}}/{{program_name}}` の形式でテンプレート変数を生成する。

4. **既存テストの更新**: `ast_transformer_test.go` と `converter_test.go` の既存テストケースが新しいロジックに合わせて更新されること。

5. **go.mod テンプレート化のテストケース追加**: 実際のユースケース（originals の `go.mod` モジュールパスと `oldModule` が異なる場合）を検証するテストケースを追加する。

### 任意要件

- `go.mod` 内の `require` ブロックにあるモジュールパスも変換対象にするかは今回のスコープ外とする。

## 実現方針 (Implementation Approach)

### 変更対象ファイル

1. **`features/templatizer/internal/converter/ast_transformer.go`**
   - `TransformGoMod` 関数のシグネチャと実装を変更
   - `oldModule` によるexact matchの判定を削除し、`go.mod` から直接 `module` 行を読み取って `newModule` に置換するロジックに変更
   - `oldModule` パラメータは不要とする（または下位互換性のために残すかは要検討）

2. **`features/templatizer/internal/converter/transform.go`**
   - `TransformGoFiles` 内の `TransformGoMod` 呼び出し箇所を新しいシグネチャに合わせて修正

3. **`features/templatizer/internal/converter/converter.go`**
   - `BuildConvertParams` でテンプレート変数の `NewModule` を `{{module_path}}/{{program_name}}` 形式に設定するか、別のアプローチで対応

4. **`features/templatizer/internal/converter/ast_transformer_test.go`**
   - 既存テストの更新と、不一致ケースのテスト追加

5. **`features/templatizer/internal/converter/converter_test.go`**
   - 全体パイプラインテストの更新

### 設計方針

`TransformGoMod` を以下のように変更する:

```go
// 変更前:
func TransformGoMod(content []byte, oldModule, newModule string) ([]byte, bool, error)
// oldModule とのexact matchが必要

// 変更後:
func TransformGoMod(content []byte, newModule string) ([]byte, bool, error)
// go.mod の module 行を無条件に newModule に置換
```

`newModule` には `{{module_path}}/{{program_name}}` 形式のテンプレート変数文字列を渡す。

また、`TransformGoFiles`から`TransformGoMod`を呼ぶ際に、`oldModule`を使わず`newModule`のみを受け取るようにする。ただし、`TransformGoSource` (`.go`ファイルのimport置換) は引き続き `oldModule` が必要なため、`TransformGoFiles` のシグネチャ自体は変更しない。`go.mod` のmoduleパスを読み取って `oldModule` として `.go` ファイルのimport変換に利用する方法も検討する。

> [!IMPORTANT]
> `TransformGoMod` が `go.mod` から実際のモジュールパスを返すようにし、それを `TransformGoSource` の `oldModule` として使う設計が望ましい。これにより `BuildConvertParams` の `OldModule` 設定の問題も解消される。

### 推奨設計: go.mod からモジュールパスを自動取得

```go
// TransformGoMod は go.mod の module 行を newModule に置換し、
// 元のモジュールパス (originalModule) を返す。
func TransformGoMod(content []byte, newModule string) (transformed []byte, originalModule string, changed bool, err error)
```

`TransformGoFiles` 内で:
1. まず `go.mod` を処理して元のモジュールパスを取得
2. そのモジュールパスを `oldModule` として `.go` ファイルの import 変換に使用

これにより、`BuildConvertParams` で `OldModule` を正しく設定できなくても、実際のソースコードから自動で取得できる。

## 検証シナリオ (Verification Scenarios)

1. `templatizer` を実行
2. `catalog/scaffolds/b/i/b/go-standard-feature.zip` を展開
3. `base/go.mod.tmpl` が存在すること (`go.mod` ではなく)
4. `go.mod.tmpl` の内容が `module {{module_path}}/{{program_name}}` であること
5. `main.go` はimport変換の対象外 (外部importも自モジュールimportもないため) なのでそのまま残ること

## テスト項目 (Testing for the Requirements)

### 単体テスト

- **テスト対象**: `TransformGoMod` 関数
  - `go.mod` の module 行が `newModule` に正しく置換されること
  - `require` ブロック等は変更されないこと
  - module 行がない場合はエラーなく未変更で返すこと

- **テスト対象**: `TransformGoFiles` 関数
  - `go.mod` ファイルが `.tmpl` にリネームされること
  - `go.mod` から取得した `oldModule` で `.go` ファイルのimportが正しく変換されること

- **テスト対象**: `Convert` 全体パイプラインテスト
  - originals の `go.mod` のモジュールパスと `oldModule` が異なる場合でも、テンプレート化が成功すること

### 検証コマンド

```bash
# ビルドとユニットテスト
scripts/process/build.sh

# 統合テスト
scripts/process/integration_test.sh
```
