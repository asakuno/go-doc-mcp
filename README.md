# Go Document MCP Server

Golangで実装されたドキュメントベクトル検索のためのMCP（Model Context Protocol）サーバーです。ドキュメントをベクトル化してPostgreSQL + pgvectorに保存し、効率的なセマンティック検索を実現します。

## 特徴

- 📚 **ドキュメントのベクトル化**: OpenAI Embeddings APIまたは **Ollama（完全無料）** を使用
- 🔍 **セマンティック検索**: コサイン類似度を使用した高度な検索機能
- 💾 **PostgreSQL + pgvector**: 高速なベクトル類似度検索をサポート
- 🔧 **MCP Protocol**: Claude Codeと統合可能なツールを提供
- 📝 **複数フォーマット対応**: .md, .txt, .go, .js, .ts, .py, .java, .c, .cpp, .h, .hpp
- 💰 **完全無料オプション**: Ollamaを使えばAPIキー不要で完全無料で利用可能

## 提供ツール

MCPサーバーは以下のツールを提供します：

1. **search_documents**: ベクトル検索でドキュメントを検索
2. **index_document**: 単一ファイルをインデックス化
3. **index_directory**: ディレクトリ内の全ファイルをインデックス化
4. **list_documents**: インデックス済みドキュメント一覧を表示
5. **delete_document**: ドキュメントを削除
6. **get_stats**: 統計情報を取得

## セットアップ

### 前提条件

- Go 1.21以上
- Docker & Docker Compose

### インストール手順

1. リポジトリをクローン:
```bash
git clone <repository-url>
cd go-doc-mcp
```

2. 依存関係をインストール:
```bash
go mod download
```

3. 環境変数を設定:
```bash
cp .env.example .env
# .envファイルを編集（無料で使うならデフォルトのままでOK）
```

4. PostgreSQLとOllamaを起動:
```bash
docker-compose up -d
```

データベースとOllamaが起動するまで待ちます：
```bash
docker-compose logs -f
# "database system is ready to accept connections"が表示されるまで待つ
```

5. Ollamaの埋め込みモデルをダウンロード（初回のみ）:
```bash
docker exec -it go-doc-mcp-ollama ollama pull nomic-embed-text
```

または、Makefileを使用:
```bash
make ollama-pull
```

### ビルドと実行

```bash
# ビルド
go build -o go-doc-mcp main.go

# 実行
./go-doc-mcp
```

または直接実行：
```bash
go run main.go
```

## 埋め込みプロバイダーの選択

### オプション1: Ollama（推奨・完全無料）

**メリット**:
- 完全無料
- APIキー不要
- プライバシー保護（データはローカルのみ）
- 高速（ローカル実行）

**設定**（デフォルト）:
```bash
EMBEDDING_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
EMBEDDING_MODEL=nomic-embed-text
EMBEDDING_DIMENSIONS=768
```

**推奨モデル**:
- `nomic-embed-text`: 768次元、バランスが良い（デフォルト）
- `mxbai-embed-large`: 1024次元、より高品質
- `all-minilm`: 384次元、軽量・高速

モデルの切り替え:
```bash
# 別のモデルをダウンロード
docker exec -it go-doc-mcp-ollama ollama pull mxbai-embed-large

# .envファイルで設定を変更
EMBEDDING_MODEL=mxbai-embed-large
EMBEDDING_DIMENSIONS=1024
```

### オプション2: OpenAI（有料）

**メリット**:
- 高品質な埋め込み
- サーバー不要

**設定**:
```bash
EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=your-api-key-here
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_DIMENSIONS=1536
```

**コスト**（2024年時点）:
- text-embedding-3-small: $0.02 / 1M tokens
- text-embedding-3-large: $0.13 / 1M tokens

## 使用方法

### Claude Codeでの設定

Claude Codeの設定ファイル（`claude_desktop_config.json`または`claude_code_config.json`）に以下を追加：

#### Ollama（無料）を使用する場合:
```json
{
  "mcpServers": {
    "go-doc-mcp": {
      "command": "/path/to/go-doc-mcp",
      "env": {
        "EMBEDDING_PROVIDER": "ollama",
        "OLLAMA_URL": "http://localhost:11434",
        "EMBEDDING_MODEL": "nomic-embed-text",
        "EMBEDDING_DIMENSIONS": "768",
        "POSTGRES_HOST": "localhost",
        "POSTGRES_PORT": "5432",
        "POSTGRES_USER": "postgres",
        "POSTGRES_PASSWORD": "postgres",
        "POSTGRES_DB": "vectordb",
        "CHUNK_SIZE": "1000",
        "CHUNK_OVERLAP": "200"
      }
    }
  }
}
```

#### OpenAI（有料）を使用する場合:
```json
{
  "mcpServers": {
    "go-doc-mcp": {
      "command": "/path/to/go-doc-mcp",
      "env": {
        "EMBEDDING_PROVIDER": "openai",
        "OPENAI_API_KEY": "your-api-key",
        "EMBEDDING_MODEL": "text-embedding-3-small",
        "EMBEDDING_DIMENSIONS": "1536",
        "POSTGRES_HOST": "localhost",
        "POSTGRES_PORT": "5432",
        "POSTGRES_USER": "postgres",
        "POSTGRES_PASSWORD": "postgres",
        "POSTGRES_DB": "vectordb"
      }
    }
  }
}
```

### 基本的な使用例

1. **ディレクトリをインデックス化**:
```
Claude Codeで: "index_directory ツールを使って ./docs ディレクトリをインデックス化して"
```

2. **ドキュメントを検索**:
```
Claude Codeで: "search_documents ツールで 'authentication' を検索して"
```

3. **統計情報を確認**:
```
Claude Codeで: "get_stats ツールでドキュメントの統計を表示して"
```

## 設定

### 環境変数

| 変数名 | 説明 | デフォルト値 |
|--------|------|--------------|
| `EMBEDDING_PROVIDER` | 埋め込みプロバイダー (ollama/openai) | `ollama` |
| `OLLAMA_URL` | Ollama APIのURL | `http://localhost:11434` |
| `OPENAI_API_KEY` | OpenAI APIキー (openai使用時のみ必須) | - |
| `EMBEDDING_MODEL` | 使用する埋め込みモデル | `nomic-embed-text` (ollama) / `text-embedding-3-small` (openai) |
| `EMBEDDING_DIMENSIONS` | 埋め込みベクトルの次元数 | `768` (ollama) / `1536` (openai) |
| `POSTGRES_HOST` | PostgreSQLホスト | `localhost` |
| `POSTGRES_PORT` | PostgreSQLポート | `5432` |
| `POSTGRES_USER` | PostgreSQLユーザー | `postgres` |
| `POSTGRES_PASSWORD` | PostgreSQLパスワード | `postgres` |
| `POSTGRES_DB` | データベース名 | `vectordb` |
| `CHUNK_SIZE` | チャンクサイズ（文字数） | `1000` |
| `CHUNK_OVERLAP` | チャンクのオーバーラップ | `200` |

### チャンクサイズの調整

- **CHUNK_SIZE**: 大きいほどコンテキストが多くなるが、精度が落ちる可能性がある
- **CHUNK_OVERLAP**: 大きいほどチャンク間の連続性が保たれる

推奨設定：
- コード: CHUNK_SIZE=800, CHUNK_OVERLAP=100
- ドキュメント: CHUNK_SIZE=1000, CHUNK_OVERLAP=200
- 長文: CHUNK_SIZE=1500, CHUNK_OVERLAP=300

## アーキテクチャ

```
┌─────────────────┐
│  Claude Code    │
└────────┬────────┘
         │ MCP Protocol
┌────────▼────────┐
│  Go MCP Server  │
│                 │
│  ┌───────────┐  │
│  │Document   │  │
│  │Loader     │  │
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │Document   │  │
│  │Chunker    │  │
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │Embedding  │◄─┼─► Ollama (無料) or OpenAI (有料)
│  │Service    │  │
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │Vector     │  │
│  │Store      │  │
│  └─────┬─────┘  │
└────────┼────────┘
         │
┌────────▼────────┐
│  PostgreSQL +   │
│  pgvector       │
└─────────────────┘
```

## プロジェクト構造

```
go-doc-mcp/
├── main.go                    # エントリーポイント
├── go.mod                     # Go modules
├── docker-compose.yml         # PostgreSQL + pgvector + Ollama設定
├── init.sql                   # データベース初期化SQL
├── .env.example              # 環境変数サンプル
├── Makefile                  # ビルド/開発タスク
├── README.md
└── internal/
    ├── mcp/
    │   ├── server.go         # MCPサーバー実装
    │   └── tools.go          # MCPツール実装
    ├── vectorstore/
    │   ├── store.go          # ベクトルストア
    │   └── embeddings.go     # 埋め込み生成（Ollama/OpenAI対応）
    └── document/
        ├── loader.go         # ドキュメント読み込み
        └── chunker.go        # テキスト分割
```

## トラブルシューティング

### PostgreSQLに接続できない

```bash
# PostgreSQLが起動しているか確認
docker-compose ps

# ログを確認
docker-compose logs postgres

# 再起動
docker-compose restart postgres
```

### Ollamaに接続できない

```bash
# Ollamaが起動しているか確認
docker-compose ps

# Ollamaのログを確認
docker-compose logs ollama

# モデルがダウンロードされているか確認
docker exec -it go-doc-mcp-ollama ollama list

# モデルを再ダウンロード
docker exec -it go-doc-mcp-ollama ollama pull nomic-embed-text
```

### 埋め込み生成でエラーが発生する

**Ollama使用時**:
- Ollamaが起動しているか確認
- モデルがダウンロードされているか確認（`ollama list`）
- OLLAMA_URLが正しいか確認

**OpenAI使用時**:
- OpenAI APIキーが正しく設定されているか確認
- APIの使用量制限に達していないか確認
- ネットワーク接続を確認

### メモリ不足エラー

- CHUNK_SIZEを小さくする
- 一度にインデックス化するファイル数を減らす
- PostgreSQLのメモリ設定を調整
- Ollamaの場合、軽量モデル（all-minilm）を使用

## パフォーマンス最適化

### Ollamaのパフォーマンス

- **nomic-embed-text**: バランス型（推奨）
- **mxbai-embed-large**: 高品質だが少し遅い
- **all-minilm**: 高速だが精度は低め

### インデックス作成

データベースには既にHNSW（Hierarchical Navigable Small World）インデックスが作成されています。
大量のドキュメントをインデックス化する場合は、以下を検討してください：

1. バッチ処理で少しずつインデックス化
2. 軽量な埋め込みモデルを使用
3. PostgreSQLの`work_mem`を増やす

### 検索速度

- limitパラメータを適切に設定（デフォルト: 5）
- 不要なドキュメントは定期的に削除

## Makefile コマンド

```bash
make help              # ヘルプ表示
make deps              # 依存関係のダウンロード
make build             # ビルド
make run               # 実行
make test              # テスト実行
make clean             # クリーンアップ
make docker-up         # Docker起動
make docker-down       # Docker停止
make docker-logs       # Dockerログ表示
make docker-reset      # Dockerデータリセット
make ollama-pull       # Ollamaモデルダウンロード
make ollama-list       # Ollamaモデル一覧
make dev               # 開発環境セットアップ
```

## よくある質問

### Q: 無料で使えますか？
A: はい！Ollamaを使用すれば完全無料で利用できます。APIキーも不要です。

### Q: どちらの埋め込みプロバイダーを選ぶべきですか？
A: 個人利用や小規模プロジェクトではOllama（無料）がおすすめです。大規模プロジェクトで最高品質が必要な場合はOpenAIを検討してください。

### Q: Ollamaの埋め込み品質はOpenAIと比べてどうですか？
A: 多くの用途では十分な品質です。特にnomic-embed-textやmxbai-embed-largeは高品質な結果を提供します。

### Q: プライバシーは大丈夫ですか？
A: Ollamaを使用すれば、すべてのデータがローカルに留まります。OpenAIを使用する場合は、データがOpenAIのサーバーに送信されます。

### Q: 既存のデータベースを別の次元数に変更できますか？
A: init.sqlの次元数を変更して、データベースを再作成する必要があります（`make docker-reset`）。

## ライセンス

MIT

## 貢献

プルリクエストを歓迎します！

## サポート

問題が発生した場合は、GitHubのIssuesでお知らせください。
