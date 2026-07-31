# 実装指示: Unix コマンド名エイリアスと未知オプションのエラー化

## 背景と目的

memsh の DSL コマンド名を Unix コマンド名に統一することで、LLM が既存の知識だけで
正しいスクリプトを組み立てられるようにする。

現状は `json.query`, `text.replace` などの独自名が残っており、SKILL.md に使用例を
明示しなければ LLM が構文を推論できない。Unix 名に統一すれば例示が不要になる。

合わせて、未知オプションが来たときにサイレントな誤動作になっている問題を修正する。
LLM はエラーメッセージを受け取って修正できるので、明示的なエラーにすることで
信頼性が上がる。

---

## 変更対象ファイル

`internal/dsl/compiler.go` のみ。

---

## 変更内容

### 1. Unix コマンド名エイリアスの追加

`compileCommand` の `switch` に以下の `case` を追加する。
既存の `compile*` 関数を再利用するだけでよい。

```go
case "jq":
    return compileJSONQuery(cmd, stdinPath, outputPath)
case "sed":
    return compileTextReplace(cmd, stdinPath, outputPath)
case "cut":
    return compileTextCut(cmd, stdinPath, outputPath)
case "wc":
    return compileTextWC(cmd, stdinPath, outputPath)
```

既存の `json.query`, `text.replace`, `text.cut`, `text.wc` の `case` はそのまま残す
（後方互換のため）。

`grep`, `sort`, `uniq`, `head`, `tail`, `cat` はすでに Unix 名なので変更不要。

---

### 2. 未知オプションの明示的エラー化

各 `compile*` 関数でオプション解析後に未処理のフラグ（`-` で始まる引数）が残って
いた場合、`unsupported_option` エラーを返すようにする。

#### `compileGrep`

現状: `-i` 以外のフラグがパターンとして解釈されてサイレントに誤動作する。

修正: `-i` の処理後、次の引数が `-` 始まりならエラーにする。

```go
func compileGrep(cmd Command, stdinPath, outputPath string) (Step, string, error) {
    step := Step{
        Tool:       "text.grep",
        OutputPath: outputPath,
        Params:     map[string]string{},
    }

    args := cmd.Args
    if len(args) >= 1 && args[0] == "-i" {
        step.Params["ignore_case"] = "true"
        args = args[1:]
    }

    // 未知オプションをエラーにする
    if len(args) >= 1 && strings.HasPrefix(args[0], "-") {
        return Step{}, "", newProtocolError("unsupported_option",
            fmt.Sprintf("grep: unsupported option %q (supported: -i)", args[0]))
    }

    if len(args) < 1 || len(args) > 2 {
        return Step{}, "", newProtocolError("missing_argument", "grep requires a pattern and optional path")
    }
    // ... 以降は既存のまま
```

#### `compileTextWC`

現状: `-l` 以外のフラグがスキップされてサイレントに無視される。

修正: `-l` の処理後、次の引数が `-` 始まりならエラーにする。

```go
    args := cmd.Args
    if len(args) >= 1 && args[0] == "-l" {
        args = args[1:]
    }

    // 未知オプションをエラーにする
    if len(args) >= 1 && strings.HasPrefix(args[0], "-") {
        return Step{}, "", newProtocolError("unsupported_option",
            fmt.Sprintf("wc: unsupported option %q (supported: -l)", args[0]))
    }
    // ... 以降は既存のまま
```

#### `compileTextCut`

現状: `-d`, `-f` 以外のフラグが `goto done` でスキップされ、その後の引数処理で
誤動作する可能性がある。

修正: ループを抜けた時点で残った args の先頭が `-` 始まりならエラーにする。

```go
done:
    // 未知オプションをエラーにする
    if len(args) >= 1 && strings.HasPrefix(args[0], "-") {
        return Step{}, "", newProtocolError("unsupported_option",
            fmt.Sprintf("cut: unsupported option %q (supported: -d, -f)", args[0]))
    }

    if step.Params["delimiter"] == "" || step.Params["fields"] == "" {
        // ... 既存のまま
```

#### `compileHeadTail`

現状: `-n` 以外のフラグが引数として扱われて誤動作する可能性がある。

修正: `-n` の処理後、次の引数が `-` 始まりならエラーにする。

```go
    args := cmd.Args
    if len(args) >= 2 && args[0] == "-n" {
        // ... 既存のまま
        args = args[2:]
    }

    // 未知オプションをエラーにする
    if len(args) >= 1 && strings.HasPrefix(args[0], "-") {
        return Step{}, "", newProtocolError("unsupported_option",
            fmt.Sprintf("%s: unsupported option %q (supported: -n)", cmd.Name, args[0]))
    }
    // ... 以降は既存のまま
```

#### `compileJSONQuery` / `compileJSONToText` / `compileTextReplace`

これらはオプションフラグを位置引数で受け取る設計（`--flat` など）のため、
`-` 始まりの未知引数チェックを同様に追加する。

`compileJSONQuery`: 引数が1〜2個という既存チェックで実質カバー済みだが、
`-` 始まりの引数が来たときのエラーメッセージをより分かりやすくする。

```go
    if len(cmd.Args) >= 1 && strings.HasPrefix(cmd.Args[0], "-") {
        return Step{}, "", newProtocolError("unsupported_option",
            fmt.Sprintf("jq: unsupported option %q (jq takes a query expression, not flags)", cmd.Args[0]))
    }
```

---

## テスト方針

既存テストが通ることを確認した上で、以下のケースを追加する。

**エイリアスのテスト** (`compiler_test.go`):
- `jq '.foo' /input/a.json > /output/b.json` が `json.query` と同じ Step に
  コンパイルされること
- `sed 's/a/b/g' /input/a.txt > /output/b.txt` が `text.replace` と同じ Step に
  コンパイルされること
- `cut -d ',' -f 1 /input/a.txt > /output/b.txt` が `text.cut` と同じ Step に
  コンパイルされること
- `wc -l /input/a.txt > /output/b.txt` が `text.wc` と同じ Step に
  コンパイルされること

**未知オプションのテスト** (`compiler_test.go`):
- `grep -v pattern /input/a.txt > /output/b.txt` が `unsupported_option` エラーに
  なること
- `wc -c /input/a.txt > /output/b.txt` が `unsupported_option` エラーになること
- `head -f /input/a.txt > /output/b.txt` が `unsupported_option` エラーになること
- `cut -z -d ',' -f 1 > /output/b.txt` が `unsupported_option` エラーになること
