# memsh への要求仕様

adk-go-sample の `run_skill_script` バックエンドとして memsh を採用するにあたり、必要な機能を整理する。

## 前提：利用シナリオ

### adk-go-sample における Skills の段階的開示

adk-go-sample は adk-python の Skills システムを Go で再現したもの。
LLM はスキルに対して以下の順でアクセスする：

1. **L1**（常時）: システムプロンプトにスキルの name + description が注入される
2. **L2**（オンデマンド）: LLM が `load_skill` を呼んで SKILL.md の手順を取得する
3. **L3**（オンデマンド）: LLM が `load_skill_resource` で assets ファイルを取得、または `run_skill_script` でスクリプト処理を実行する

memsh は **L3 の `run_skill_script`** のバックエンドとして使う。

### 想定フロー（solution_map スキルの例）

1. LLM が `load_skill("solution-map-context")` を呼ぶ
2. SKILL.md に「`scripts/query.py` でデータを取得せよ」と書いてある
3. LLM が `run_skill_script("solution-map-context", "query.py", ["--unit", "map"])` を呼ぶ
4. adk-go-sample が memsh セッションを開き、`assets/data.json` を `/input/data.json` としてロード
5. 引数を memsh DSL に変換して `Execute()` する
6. 結果テキストを LLM に返す

### データの性質

- `assets/data.json` は構造化 JSON（数百〜数千行規模）
- LLM は「特定のマップの課題エントリだけ取り出す」「キーワードで横断検索する」などの操作をしたい
- 結果はテキストで LLM に返す

---

## 要求 1: JSON 操作コマンド（最優先）

### なぜ必要か

**LLM がデータの必要な部分だけを取り出せるようにするため。**

現在の solution_map スキルは、Go コードが JSON 全体をパースして必要なエントリだけを抽出し、テキストに整形して LLM に返している。
これはコードが増えるほど保守コストが高く、新しいクエリパターンへの対応も都度開発が必要になる。

memsh に `json.query` があれば、LLM 自身が「どのフィールドをどう絞り込むか」を DSL で表現できる。
Go 側はデータを `/input/` に渡すだけでよくなり、クエリロジックをコードとして持つ必要がなくなる。

また、JSON データはスキルの assets として置くのが自然な設計だが、
JSON のままでは LLM のコンテキストに丸ごと載せるには大きすぎることが多い。
`json.query` で必要な部分だけ取り出してから LLM に返すことで、**L3 の「必要なときに必要なだけ取得する」**という段階的開示の設計意図を実現できる。

### 必要なコマンド

#### `json.query`

jq 相当のフィルタリング。

```
json.query '<jq_expression>' /input/data.json > /output/result.json
```

- jq 互換の式をサポートする（`.entries[] | select(.label == "foo")` など）
- 出力は JSON テキスト
- Go 実装として [itchyny/gojq](https://github.com/itchyny/gojq)（jq 互換ライブラリ）の利用を推奨

#### `json.to_text`

JSON を人間が読みやすいテキスト（インデント整形）に変換。

```
json.to_text /input/data.json > /output/result.txt
```

**なぜ必要か**: `json.query` の結果は JSON のままであることが多い。
LLM はテキストとして受け取った方が処理しやすく、また LLM への返却コンテンツとしてもテキストの方が自然。
`json.query | json.to_text` のパイプラインで「絞り込み → 整形」を一連で行えるようにしたい。

---

## 要求 2: テキスト変換コマンド

### `text.replace`

文字列の置換。

```
text.replace 's/old/new/g' /input/file.txt > /output/result.txt
```

- 正規表現対応
- sed の `s/pattern/replacement/flags` 構文

**なぜ必要か**: JSON や構造化テキストをそのまま LLM に返すと、不要なノイズ（記号・フォーマット）が含まれることがある。
LLM への返却前に整形・クリーニングするための最後の一手として使いたい。
また、スキルの assets に URL テンプレートや定型文が含まれる場合、動的な値に置換して使うユースケースがある。

### `text.cut`

フィールド抽出。

```
text.cut -d ',' -f 1,3 /input/data.csv > /output/result.txt
```

- 区切り文字と列番号を指定

**なぜ必要か**: スキルの assets が CSV 形式の場合、全列を LLM に渡すとコンテキストが膨らむ。
必要な列だけ抽出して渡すことで、L3 の段階的開示の趣旨を CSV でも実現できる。

### `text.wc`

行数・文字数カウント。

```
text.wc -l /input/file.txt > /output/count.txt
```

**なぜ必要か**: LLM がデータの規模感を把握するために使う。
「このファイルには何件のエントリがあるか」を先に確認してから詳細クエリを組み立てるという、LLM の自律的なデータ探索を支援できる。

---

## 要求 3: 複数回 Execute のサポート

### なぜ必要か

**LLM はデータを一度で把握できないため、段階的に問い合わせる必要があるから。**

現状、`Execute` は1セッションにつき1回しか呼べない。
しかし LLM との対話では、次のような多段クエリが自然に発生する：

- ステップ1: `json.query '.maps[].name'` → マップ名の一覧を取得
- ステップ2: 特定のマップ名が分かった上で `json.query '.entries[] | select(.map == "AIプロダクト提案")'` → 詳細を取得

ステップ1の結果を見てからステップ2のクエリを組み立てる、という流れは LLM が自律的に動くときの典型パターン。
これが1回の `Execute` に収まるとは限らない。

`Execute` を複数回呼べるようにすることで、1つの `run_skill_script` の呼び出しの中で LLM が試行錯誤できるようになる。

### 要求

- 同一セッション内で `Execute` を複数回呼べるようにしてほしい
- 前の `Execute` の `/output/` が次の `Execute` でも参照できると望ましい
- あるいは `/work/` への書き込みが次の `Execute` で引き続き参照できる設計でもよい

---

## 要求 4: エラー出力の構造化

### なぜ必要か

**LLM がエラーの原因を理解して自己修正できるようにするため。**

現在の `broker.Result` は成功時の出力（`Output []byte`）しか持っていない。
コマンドが失敗した場合、Go 側は error を受け取るが、「どのステップで何が原因で失敗したか」の詳細が LLM に伝わらない。

LLM は試行錯誤しながらクエリを組み立てる。jq 式が構文エラーだった、ファイルパスが間違っていた、といった失敗の詳細が返れば LLM は自分でクエリを修正して再試行できる。
エラー詳細がなければ LLM は「失敗した」という事実しかわからず、回復不能になる。

### 要求

`Result` に以下を追加してほしい：

```go
type Result struct {
    OutputPath string
    Output     []byte
    Stderr     []byte  // 追加: エラーメッセージ・警告
    ExitCode   int     // 追加: 0=成功、非0=失敗
}
```

---

## 優先度まとめ

| 要求 | 優先度 | 理由 |
|---|---|---|
| `json.query`（jq 相当） | 最高 | solution_map の JSON クエリに必須。これがないと memsh 採用の価値が薄い |
| 複数回 Execute | 高 | LLM の多段クエリに必要。1回制限では自律的なデータ探索ができない |
| エラー出力の構造化 | 高 | LLM の自己修正ループに必要。エラー詳細がないと回復不能になる |
| `json.to_text` | 中 | `json.query` とセットで使う整形ステップ |
| `text.replace` | 中 | 返却テキストのクリーニング・テンプレート展開に使う |
| `text.cut` | 低 | CSV 形式の assets を扱うスキルで使う |
| `text.wc` | 低 | LLM のデータ規模把握を支援する |
