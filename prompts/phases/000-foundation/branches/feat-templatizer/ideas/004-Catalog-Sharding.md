# catalog.yaml シャーディング: scaffold 定義のファイル分割

## 背景 (Background)

`catalog.yaml` は現在、すべての scaffold エントリを単一ファイルに一元管理している。scaffold 数が増加するとファイルが大きくなり、以下の問題が生じる：

- **ファイルサイズの増大**: scaffold が数十〜数百に増えると、YAML ファイルが肥大化し管理しにくくなる
- **アクセス効率**: クライアント（`tt` CLI）が特定の scaffold を取得するためだけに全体をダウンロードする必要がある
- **GitHub Contents API のコスト**: API 呼び出しごとにファイル全体を取得するのは非効率

### 解決アプローチ

**ハッシュベースシャーディング**（Git のオブジェクトストレージと同様）により、各 scaffold 定義を個別ファイルに分割する。クライアントは `category` と `name` からハッシュを算出し、該当ファイルに直接アクセスできるため、インデックスファイルの取得は不要となる。

## 要件 (Requirements)

### 必須要件

#### R1: FNV-1a 32ビット + base-36 エンコードによるハッシュパス

`category + "/" + name` を FNV-1a 32ビットでハッシュし、36^4（1,679,616）で剰余を取った後、base-36（`0-9a-z`）で4文字にエンコードする。1文字ずつ区切ってディレクトリ階層を構成する。

**ハッシュ計算:**

```go
import (
    "hash/fnv"
    "strconv"
    "fmt"
)

func scaffoldHash(category, name string) string {
    h := fnv.New32a()
    h.Write([]byte(category + "/" + name))
    v := h.Sum32() % 1679616  // 36^4
    s := strconv.FormatUint(uint64(v), 36)
    return fmt.Sprintf("%04s", s)  // 0パディングで4文字固定
}

// 例: "feature/axsh-go-standard" → "a3k9"
// パス: catalog/scaffolds/a/3/k/9.yaml
```

**ディレクトリ構造:**

```
catalog/
├── catalog.yaml              ← 最小メタデータ（version, default_scaffold, updated_at）
└── scaffolds/
    ├── a/
    │   └── 3/
    │       └── k/
    │           └── 9.yaml    ← scaffold 定義（配列形式）
    ├── b/
    │   └── 7/
    │       └── ...
    └── ...
```

**パス構造のスペック:**

| 項目 | 値 |
|---|---|
| ハッシュ関数 | FNV-1a 32ビット |
| 名前空間 | 36^4 = 1,679,616 |
| エンコード | base-36（`0-9a-z`） |
| 文字数 | 4文字固定（0パディング） |
| ディレクトリ分割 | 1文字ずつ（3階層 + 1文字ファイル名） |
| 各階層の最大エントリ数 | 36 |

#### R2: 個別 scaffold YAML のフォーマット（配列形式）

各シャーディングファイルは、ハッシュ衝突時に複数エントリを格納できるよう **配列形式** とする。

```yaml
# catalog/scaffolds/a/3/k/9.yaml
scaffolds:
  - name: "axsh-go-standard"
    category: "feature"
    description: "AXSH Go Standard Feature"
    depends_on:
      - category: "project"
        name: "axsh-go-standard"
    template_ref: "catalog/templates/axsh/go-standard-feature"
    original_ref: "catalog/originals/axsh/go-standard-feature"
    placement_ref: "catalog/placements/axsh/go-standard-feature.yaml"
    requirements:
      directories: ["features"]
      files: []
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        default: "github.com/axsh/tokotachi/features/myprog"
      - name: "program_name"
        description: "Program name"
        required: true
        default: "myprog"
  # ハッシュ衝突した場合、ここに別のエントリが追加される
```

クライアントはファイル取得後、配列から `category` + `name` で目的のエントリをフィルタする。

#### R3: `catalog.yaml` の最小化

シャーディング後の `catalog.yaml` は最小メタデータのみとする。

```yaml
version: "1.0.0"
default_scaffold: "default"
updated_at: "2026-03-10T16:00:00+09:00"   # キャッシュ判定用タイムスタンプ
```

- `updated_at`: templatizer 実行時に自動更新される。クライアントはこの値でローカルキャッシュの有効性を判断できる
- scaffold 定義の詳細はシャーディングファイルに移動

#### R4: templatizer によるシャーディングファイル生成

templatizer の処理フローに、scaffold 定義のシャーディングファイル出力を追加する。

**処理フロー:**

```
templatizer <catalog.yaml>
     │
     ├─ catalog.yaml を読み込み
     ├─ scaffolds エントリごとに:
     │   ├─ テンプレート変換パイプライン（既存処理）
     │   └─ ZIP 生成（既存処理）
     ├─ シャーディングファイル生成:          ← 新規追加
     │   ├─ 各 scaffold のハッシュを算出
     │   ├─ ハッシュごとにグルーピング
     │   ├─ catalog/scaffolds/{hash[0]}/{hash[1]}/{hash[2]}/{hash[3]}.yaml に出力
     │   └─ 既存のシャーディングファイルをクリーンアップ
     ├─ catalog.yaml を最小メタデータで上書き
     │   └─ updated_at を現在時刻に更新
     └─ 完了
```

### 任意要件

#### O1: ハッシュ衝突の統計表示

templatizer 実行時に、衝突が発生したハッシュの一覧を表示する（デバッグ・確認用）。

## 実現方針 (Implementation Approach)

### 変更対象

```
catalog.yaml                                           # 最小メタデータに縮小
catalog/scaffolds/                                     # 新ディレクトリ（シャーディングファイル配置先）
features/templatizer/internal/catalog/catalog.go       # ハッシュ関数、シャーディング関連の型・関数追加
features/templatizer/internal/catalog/catalog_test.go  # シャーディング関連のテスト追加
features/templatizer/main.go                           # シャーディング出力処理の追加
```

### 設計詳細

#### 1. ハッシュ関数の実装

```go
// catalog.go に追加

// ScaffoldHash は category/name から 4文字の base-36 ハッシュ文字列を返す
func ScaffoldHash(category, name string) string {
    h := fnv.New32a()
    h.Write([]byte(category + "/" + name))
    v := h.Sum32() % 1679616  // 36^4
    s := strconv.FormatUint(uint64(v), 36)
    return fmt.Sprintf("%04s", s)
}

// ScaffoldShardPath はハッシュから scaffolds ディレクトリ内の相対パスを返す
func ScaffoldShardPath(hash string) string {
    return fmt.Sprintf("catalog/scaffolds/%c/%c/%c/%c.yaml", hash[0], hash[1], hash[2], hash[3])
}
```

#### 2. シャーディング出力処理

`main.go` で、全 scaffold の処理後にシャーディングファイルを生成する：

1. 各 scaffold の `category + "/" + name` からハッシュを算出
2. 同じハッシュの scaffold をグルーピング
3. グループごとに YAML ファイルを出力
4. `catalog.yaml` を `version` + `default_scaffold` + `updated_at` のみに更新

### 影響範囲

- **templatizer**: シャーディング出力処理の追加。既存の ZIP 生成処理には影響なし
- **catalog.yaml**: scaffold 詳細が削除され最小メタデータのみになる（破壊的変更）
- **tt CLI**: scaffold 取得ロジックをハッシュベースに変更する必要あり（本仕様のスコープ外、将来対応）
- **リファレンスマニュアル**: シャーディング構造とハッシュ計算式の追記が必要

> [!IMPORTANT]
> 本仕様の実装中は、`catalog.yaml` に従来通り scaffold 定義を記述し続ける（入力用）。templatizer がシャーディングファイルを生成する際に `catalog.yaml` を読み込み、出力としてシャーディングファイルと最小化された `catalog.yaml` を生成する。つまり、`catalog.yaml` は「入力（フル定義）」と「出力（最小メタデータ）」の2つの状態を持つ。

## 検証シナリオ (Verification Scenarios)

### シナリオ1: ハッシュ計算の確認

1. 既知の `category/name` ペアに対して `ScaffoldHash` を呼び出す
2. 期待される 4文字 base-36 文字列が返ることを確認する
3. 同じ入力に対して常に同じハッシュが返ること（冪等性）を確認する

### シナリオ2: シャーディングファイル生成

1. templatizer を実行する
2. `catalog/scaffolds/` 以下に YAML ファイルが生成されることを確認する
3. 各ファイルが `scaffolds` 配列形式の YAML であることを確認する
4. 元の `catalog.yaml` のすべての scaffold が、いずれかのシャーディングファイルに存在することを確認する

### シナリオ3: catalog.yaml の最小化

1. templatizer 実行後の `catalog.yaml` に `version`, `default_scaffold`, `updated_at` のみが含まれることを確認する
2. `updated_at` が実行時刻で更新されていることを確認する

### シナリオ4: ハッシュ衝突時の配列格納

1. ハッシュが衝突する2つの scaffold エントリを用意する（テスト用にハッシュ関数をモックするか、衝突するペアを探す）
2. 同一ファイルに複数エントリが配列として格納されることを確認する

## テスト項目 (Testing for the Requirements)

### 単体テスト

| テスト対象 | テスト内容 | テストファイル |
|---|---|---|
| `ScaffoldHash` | 既知の入力に対して正しいハッシュが返ること | `internal/catalog/catalog_test.go` |
| `ScaffoldHash` | 冪等性（同じ入力 → 同じ出力） | `internal/catalog/catalog_test.go` |
| `ScaffoldHash` | 結果が常に4文字であること（0パディング） | `internal/catalog/catalog_test.go` |
| `ScaffoldShardPath` | 正しいファイルパスが返ること | `internal/catalog/catalog_test.go` |
| シャーディング出力 | 全 scaffold がいずれかのファイルに含まれること | `internal/catalog/catalog_test.go` |

### 検証コマンド

```bash
# ビルド＆ユニットテスト
./scripts/process/build.sh
```
