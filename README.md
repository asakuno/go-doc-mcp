# Go Document MCP Server

Golangで実装されたドキュメントベクトル検索のためのMCP（Model Context Protocol）サーバーです。ドキュメントをベクトル化してPostgreSQL + pgvectorに保存し、効率的なセマンティック検索を実現します。

## 特徴

- 📚 **ドキュメントのベクトル化**: OpenAI Embeddings APIを使用してドキュメントをベクトル化
- 🔍 **セマンティック検索**: コサイン類似度を使用した高度な検索機能
- 💾 **PostgreSQL + pgvector**: 高速なベクトル類似度検索をサポート
- 🔧 **MCP Protocol**: Claude Codeと統合可能なツールを提供
- 📝 **複数フォーマット対応**: .md, .txt, .go, .js, .ts, .py, .java, .c, .cpp, .h, .hpp

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
- OpenAI API Key

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
# .envファイルを編集してOpenAI APIキーなどを設定
```

必須の環境変数：
```bash
OPENAI_API_KEY=your-api-key-here
```

4. PostgreSQL + pgvectorを起動:
```bash
docker-compose up -d
```

データベースが起動するまで待ちます：
```bash
docker-compose logs -f postgres
# "database system is ready to accept connections"が表示されるまで待つ
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

## 使用方法

### Claude Codeでの設定

Claude Codeの設定ファイル（`claude_desktop_config.json`または`claude_code_config.json`）に以下を追加：

```json
{
  "mcpServers": {
    "go-doc-mcp": {
      "command": "/path/to/go-doc-mcp",
      "env": {
        "OPENAI_API_KEY": "your-api-key",
        "POSTGRES_HOST": "localhost",
        "POSTGRES_PORT": "5432",
        "POSTGRES_USER": "postgres",
        "POSTGRES_PASSWORD": "postgres",
        "POSTGRES_DB": "vectordb",
        "CHUNK_SIZE": "1000",
        "CHUNK_OVERLAP": "200",
        "EMBEDDING_MODEL": "text-embedding-3-small",
        "EMBEDDING_DIMENSIONS": "1536"
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
| `POSTGRES_HOST` | PostgreSQLホスト | `localhost` |
| `POSTGRES_PORT` | PostgreSQLポート | `5432` |
| `POSTGRES_USER` | PostgreSQLユーザー | `postgres` |
| `POSTGRES_PASSWORD` | PostgreSQLパスワード | `postgres` |
| `POSTGRES_DB` | データベース名 | `vectordb` |
| `OPENAI_API_KEY` | OpenAI APIキー | *必須* |
| `CHUNK_SIZE` | チャンクサイズ（文字数） | `1000` |
| `CHUNK_OVERLAP` | チャンクのオーバーラップ | `200` |
| `EMBEDDING_MODEL` | 使用する埋め込みモデル | `text-embedding-3-small` |
| `EMBEDDING_DIMENSIONS` | 埋め込みベクトルの次元数 | `1536` |

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
│  │Embedding  │◄─┼─► OpenAI API
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
├── docker-compose.yml         # PostgreSQL + pgvector設定
├── init.sql                   # データベース初期化SQL
├── .env.example              # 環境変数サンプル
├── README.md
└── internal/
    ├── mcp/
    │   ├── server.go         # MCPサーバー実装
    │   └── tools.go          # MCPツール実装
    ├── vectorstore/
    │   ├── store.go          # ベクトルストア
    │   └── embeddings.go     # 埋め込み生成
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

### 埋め込み生成でエラーが発生する

- OpenAI APIキーが正しく設定されているか確認
- APIの使用量制限に達していないか確認
- ネットワーク接続を確認

### メモリ不足エラー

- CHUNK_SIZEを小さくする
- 一度にインデックス化するファイル数を減らす
- PostgreSQLのメモリ設定を調整

## パフォーマンス最適化

### インデックス作成

データベースには既にHNSW（Hierarchical Navigable Small World）インデックスが作成されています。
大量のドキュメントをインデックス化する場合は、以下を検討してください：

1. バッチ処理で少しずつインデックス化
2. `EMBEDDING_MODEL`を`text-embedding-3-small`（より高速）に変更
3. PostgreSQLの`work_mem`を増やす

### 検索速度

- limitパラメータを適切に設定（デフォルト: 5）
- 不要なドキュメントは定期的に削除

## ライセンス

MIT

## 貢献

プルリクエストを歓迎します！

## サポート

問題が発生した場合は、GitHubのIssuesでお知らせください。
