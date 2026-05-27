# テンプレートパラメータ値型定義 (Value Specification)

## 背景 (Background)

現在、`scaffold.yaml` の `template_params` には以下のフィールドのみが定義されている：

```yaml
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myfunction"
```

Go構造体（`catalog.go` の `TemplateParam`）としても同様：

```go
type TemplateParam struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Required    bool   `yaml:"required"`
    Default     string `yaml:"default,omitempty"`
    OldValue    string `yaml:"old_value"`
}
```

このためユーザーが任意の値を入力でき、バリデーションが一切行われない。不正な値（空文字、極端に長い文字列、期待される形式と異なる入力など）がそのまま処理されてしまう問題がある。

## 要件 (Requirements)

### 必須要件

1. **型定義（type）**: 各パラメータに型を指定できること
   - `string`: 文字列型（デフォルト）
   - `number`: 数字型（整数）

2. **長さ制約（length）**: パラメータ値の長さを制約できること
   - `max_bytes`: 最大バイト数
   - `max_chars`: 最大文字数（ルーン数）
   - `max_digits`: 最大桁数（`number` 型の場合）

3. **フォーマット制約（format）**: 正規表現パターンによるフォーマット検証
   - `pattern`: Go の `regexp` パッケージ互換の正規表現文字列

4. **範囲制約（range）**: 数値や文字列長の範囲指定
   - **採用形式: JSONSchema風の `minimum` / `maximum` / `exclusive_minimum` / `exclusive_maximum`**
   - 理由: 評価式（`>= 1 && <= 100` のような自由記述）は解析が複雑でセキュリティリスクもある。JSONSchema風のキーワードベースの制約は宣言的で、パースが容易かつ安全

5. **enum制約**: 許可される値を列挙できること
   - `enum: ["value1", "value2"]`

### デフォルト挙動 (templatizer による自動補完)

6. `templatizer` が未定義パラメータを自動追加する際（`MergeParams` 内の auto-add）、以下のデフォルト値型を付与する：
   - `type: string`
   - `length.max_bytes: 256`

7. `scaffold.yaml` に既に `value_spec` が記載されているパラメータはそのまま尊重する（上書きしない）

## 実現方針 (Implementation Approach)

### YAML形式の設計

`template_params` に `value_spec` フィールドを追加する：

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

  - name: "port_number"
    description: "Server port number"
    required: false
    default: "8080"
    value_spec:
      type: number
      range:
        minimum: 1
        maximum: 65535
```

### Go構造体の変更

```go
// ValueSpec defines validation rules for a template parameter.
type ValueSpec struct {
    Type   string      `yaml:"type,omitempty"`   // "string" or "number"
    Length *LengthSpec `yaml:"length,omitempty"`
    Format *FormatSpec `yaml:"format,omitempty"`
    Range  *RangeSpec  `yaml:"range,omitempty"`
}

// LengthSpec defines length constraints.
type LengthSpec struct {
    MaxBytes  *int `yaml:"max_bytes,omitempty"`
    MaxChars  *int `yaml:"max_chars,omitempty"`
    MaxDigits *int `yaml:"max_digits,omitempty"`
}

// FormatSpec defines format constraints.
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

// TemplateParam represents a single template conversion parameter.
type TemplateParam struct {
    Name        string     `yaml:"name"`
    Description string     `yaml:"description"`
    Required    bool       `yaml:"required"`
    Default     string     `yaml:"default,omitempty"`
    OldValue    string     `yaml:"old_value"`
    ValueSpec   *ValueSpec `yaml:"value_spec,omitempty"`
}
```

### 主要変更箇所

| コンポーネント | ファイル | 変更内容 |
|---|---|---|
| データモデル | `features/templatizer/internal/catalog/catalog.go` | `ValueSpec` 関連構造体の追加、`TemplateParam` への `ValueSpec` フィールド追加 |
| パラメータマージ | `features/templatizer/internal/converter/param_collector.go` | `MergeParams` の auto-add 時にデフォルト `ValueSpec` を付与 |
| scaffold.yaml | `catalog/originals/*/scaffold.yaml` | 各scaffold.yaml への `value_spec` 記載（既存パラメータへの追加） |
| テスト | `features/templatizer/internal/catalog/catalog_test.go` | `ValueSpec` のパース・シリアライズテスト追加 |
| テスト | `features/templatizer/internal/converter/param_collector_test.go` | `MergeParams` デフォルト `ValueSpec` 付与テスト追加 |

### range に JSONSchema スタイルを採用した理由

ユーザーが「範囲の良い表現形式」を質問しているので、以下に比較検討を示す：

| 方式 | 例 | 利点 | 欠点 |
|---|---|---|---|
| **JSONSchema風（採用）** | `minimum: 1, maximum: 100` | 宣言的で安全。パース容易。業界標準 | 複雑な条件（倍数制約等）は別途拡張が必要 |
| 評価式 | `">= 1 && <= 100"` | 柔軟 | パーサ実装が複雑。コードインジェクションのリスク |
| 区間表記 | `"[1, 100]"` | 数学的に直感的 | 文字列パースが必要。排他/包含の表記が紛らわしい |
| CEL式 | `"value >= 1 && value <= 100"` | Googleのポリシー言語。強力 | 依存ライブラリが重い。Learning curveが高い |

JSONSchema風は Kubernetes CRD Validation、OpenAPI Specification などで広く使われており、Go エコシステムとの親和性が高い。

## 検証シナリオ (Verification Scenarios)

### シナリオ1: scaffold.yaml に value_spec を記載してパースできること
1. `scaffold.yaml` に `value_spec` フィールドを追加した YAML を用意する
2. `ParseScaffoldDefinition()` でパースする
3. `ValueSpec` の各フィールド（type, length, format, range）が正しく読み取れることを確認

### シナリオ2: MergeParams で auto-add 時にデフォルト ValueSpec が付与されること
1. scaffold.yaml には `module_path` のみ定義（value_spec あり）
2. テンプレートから `feature_name` も発見される
3. `MergeParams` 実行後、`feature_name` に `type: string, length.max_bytes: 256` のデフォルト ValueSpec が付与されている
4. `module_path` の元の ValueSpec は変更されていない

### シナリオ3: 既存 scaffold.yaml の value_spec が尊重されること
1. scaffold.yaml に `value_spec` が記載されたパラメータがある
2. `MergeParams` で同パラメータが discovered に含まれる
3. 元の `value_spec` がそのまま保持される（上書きされない）

### シナリオ4: catalog.yaml への出力に value_spec が含まれること
1. templatizer を実行して catalog.yaml を生成する
2. 生成された catalog.yaml / shard ファイル内の `template_params` に `value_spec` が含まれていることを確認

## テスト項目 (Testing for the Requirements)

### 自動テスト

| 要件 | テスト種別 | 検証コマンド |
|---|---|---|
| ValueSpec 構造体パースの正確性 | 単体テスト | `scripts/process/build.sh` |
| MergeParams デフォルト付与 | 単体テスト | `scripts/process/build.sh` |
| scaffold.yaml 既存 value_spec の尊重 | 単体テスト | `scripts/process/build.sh` |
| templatizer 実行結果への反映 | 統合テスト | `scripts/process/integration_test.sh` |
| 全体ビルドの成功 | ビルド確認 | `scripts/process/build.sh` |
