# jbuntai

日本語テキストを情報文体に変換するCLIツール。

## 概要

`jbuntai` は日本語テキストを「情報文体」に変換します。情報文体とは、不要な文法要素を除去し、助詞を記号に置き換えることで簡潔に表現する記法です。

**変換前：**

```
本日の会議では、新製品の開発スケジュールについて検討を行いました。
```

**変換後（ルールベース）：**

```
本日の会議で、新製品の開発スケジュールについて検討行い。
```

**変換後（LLM 使用時、デフォルト）：**

```
@本日の会議 新製品の開発スケジュール検討。
```

## 情報文体とは

情報文体は、日本語テキストの情報密度を上げ、より速く情報を取得できるようにする記法です。

### 基本原則

1. **意味語は保持** — 名詞、動詞語幹、形容詞、副詞
2. **機能語は省略・記号化** — 助詞、助動詞、丁寧表現
3. **復元可能性を維持** — 文脈から元の意味が推測できること

### 省略ルール

| カテゴリ  | 対象    | 処理               |
|-------|-------|------------------|
| 丁寧語尾  | です、ます | 削除               |
| 断定語尾  | だ、である | 削除               |
| 主格助詞  | は、が   | 削除               |
| 目的格助詞 | を     | 複合語化できる場合に削除     |
| 形式名詞  | こと、もの | 削除               |
| 連体助詞  | の     | 保持（漢字連続時の境界マーカー） |

### 記号化ルール

| 記号  | 意味    | 置換対象     | 例          |
|-----|-------|----------|------------|
| `>` | 方向・対象 | に、へ      | `>東京 行く`   |
| `@` | 場所    | で（場所）    | `@会議室 検討`  |
| `→` | 結果・因果 | （因果連鎖）   | `A→B`       |
| `∵` | 理由    | ので、ため    | `∵予算不足 延期` |
| `,` | 動作連結  | て、で      | `調べ,報告`    |
| `:` | 引用    | と（言う/思う） | `:重要 判断`   |
| `〜` | 範囲    | から...まで  | `1月〜3月`    |
| `・` | 並列    | と、や      | `犬・猫`      |

## インストール

### 前提条件

LLM 変換（デフォルト有効）を使うには、以下のいずれかが必要です：

- **codex バックエンド（デフォルト）**: ChatGPT サブスクリプション（Plus/Pro/Team）。初回実行時にブラウザで OAuth 認証を行います。
- **claudecode バックエンド**: [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) がインストール・認証済みであること。

### go install

```bash
go install github.com/chez-shanpu/jbuntai@latest
```

### ソースからビルド

```bash
git clone https://github.com/chez-shanpu/jbuntai.git
cd jbuntai
make build
# バイナリ: bin/jbuntai
```

## 使い方

```bash
# 標準入力からパイプ（デフォルトで LLM 変換が有効）
echo "本日の会議では検討を行いました。" | jbuntai

# ファイル入力
jbuntai document.md

# 複数ファイル
jbuntai file1.md file2.md

# ルールベースのみで変換（LLM を無効化）
echo "本日の会議では検討を行いました。" | jbuntai --llm=false

# Claude Code CLI バックエンドを使用
echo "本日の会議では検討を行いました。" | jbuntai --backend claudecode

# 出力先をファイルに指定
jbuntai -o output.md document.md

# 圧縮率の表示
jbuntai --stats doc.md
```

## オプション

| フラグ              | 説明                                                |
|------------------|---------------------------------------------------|
| `--backend`      | LLM バックエンド: `codex`（デフォルト）または `claudecode`        |
| `--llm`          | LLM を使った変換（デフォルト: 有効）。`--llm=false` で無効化          |
| `--stats`        | 圧縮率の統計情報を標準エラー出力に表示                               |
| `--output`, `-o` | 出力先ファイルパス（デフォルト: 標準出力）                            |
| `--config パス`    | 設定ファイルのパス（デフォルト: `~/.config/jbuntai/config.yaml`） |
| `--debug`        | デバッグログをタイムスタンプ付きで標準エラー出力に表示                       |

## LLM 連携

`jbuntai` はデフォルトで LLM を使い、変換品質を向上させます。LLM には2つの機能があります：

- **曖昧性解消（Disambiguation）**: 軽量モデルで助詞の分類が曖昧なケースを解決します。
- **仕上げ（Finishing）**: 高性能モデルでルールベースの出力をより自然な情報文体に再構成します。

### バックエンド

| バックエンド | 説明 | デフォルトモデル |
|-----------|------|---------------|
| `codex`（デフォルト） | ChatGPT Responses API（OAuth PKCE 認証） | 曖昧性解消: `gpt-5.4-mini`, 仕上げ: `gpt-5.4` |
| `claudecode` | Claude Code CLI (`claude -p`) | 曖昧性解消: `haiku`, 仕上げ: `sonnet` |

### codex バックエンド（デフォルト）

ChatGPT サブスクリプション（Plus/Pro/Team）のアカウントで LLM を利用します。

1. そのまま実行します（初回はブラウザで認証が開きます）：

   ```bash
   echo "会議室で検討を行いました。" | jbuntai
   ```

2. トークンは `~/.config/jbuntai/codex_token.json` に保存され、以降は自動で再利用・リフレッシュされます。

### claudecode バックエンド

[Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) を使用します。

1. Claude Code CLI をインストール・認証します。

2. `--backend claudecode` を指定して実行します：

   ```bash
   echo "会議室で検討を行いました。" | jbuntai --backend claudecode
   ```

### LLM を無効化する場合

```bash
echo "会議室で検討を行いました。" | jbuntai --llm=false
```

### LLM の設定

`config.yaml` でモデルと各機能のオン・オフを設定できます：

```yaml
llm:
  backend: "codex"               # "codex" or "claudecode"
  disambiguate_model: ""         # 空 = バックエンドのデフォルト
  finish_model: ""               # 空 = バックエンドのデフォルト
  disambiguate: true             # 曖昧性解消の有効/無効
  finish: true                   # 仕上げの有効/無効
```

- `disambiguate: false` や `finish: false` で個別のステップを無効化できます。
- `--llm=false` を指定した場合、設定に関係なく LLM 呼び出しは行われません。
- LLM 呼び出しが失敗した場合、ルールベースの結果に自動でフォールバックします。

## 設定

設定ファイルの場所: `~/.config/jbuntai/config.yaml`

```yaml
max_kanji_run: 5
llm:
  backend: "codex"
  disambiguate: true
  finish: true
```

| キー | 説明 | デフォルト |
|------|------|-----------|
| `max_kanji_run` | 境界（スペース）を挿入するまでの連続漢字の最大数 | `5` |
| `llm.backend` | LLM バックエンド（`"codex"` または `"claudecode"`） | `"codex"` |
| `llm.disambiguate_model` | 曖昧性解消に使うモデル（空文字 = バックエンドのデフォルト） | `""` |
| `llm.finish_model` | 仕上げに使うモデル（空文字 = バックエンドのデフォルト） | `""` |
| `llm.disambiguate` | LLM 有効時に曖昧性解消を有効化 | `true` |
| `llm.finish` | LLM 有効時に仕上げを有効化 | `true` |

## アーキテクチャ

```
入力テキスト
  │
  ├─ 前処理（コードブロック・Markdown の保護）
  │
  ├─ 形態素解析（kagome + IPA辞書）
  │
  ├─ パス（ルールベース変換）:
  │    ├─ EndingPass    — 冗長な語尾の除去（ます→, です→）
  │    ├─ SymbolPass    — 助詞を記号に置換（で→@, に→>）
  │    ├─ DeletionPass  — 不要な助詞の削除（を, は 等）
  │    └─ BoundaryPass  — 長い漢字連続へのスペース挿入
  │
  ├─ LLM 仕上げ（デフォルト有効, --llm=false で無効化）
  │
  └─ 後処理（保護ブロックの復元）
         │
      出力テキスト
```

## 開発

```bash
make build       # bin/jbuntai にバイナリをビルド
make test        # staticcheck + ユニットテスト実行
make test-e2e    # E2Eテスト実行
make fmt         # コードフォーマット（gofumpt + goimports）
make vet         # go vet 実行
make check       # vet + check-diff + test
make check-all   # check + E2Eテスト
make clean       # ビルド成果物の削除
```
