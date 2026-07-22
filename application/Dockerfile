# ==========================================
# 1. ビルド用のステージ (Build Stage)
# ==========================================
FROM golang:1.26.4-alpine AS builder

# ワークディレクトリの設定
WORKDIR /app

# モジュール管理ファイルがある場合はコピーしてダウンロード
# (今回はシンプルな構成なので、なければスルーされます)
COPY go.mod* go.sum* ./
RUN if [ -f go.mod ]; then go mod download; fi

# ソースコードとカタログ編集元をコピー
COPY . .

# YAML から読み取り専用カタログを生成
RUN go run ./cmd/catalog-build -source ./catalog/mangas -output /app/catalog.db

# Goバイナリをビルド (CGOを無効化し、Linux向けに最適化)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

# ==========================================
# 2. 実行用のステージ (Run Stage)
# ==========================================
FROM alpine:latest

WORKDIR /app

# ビルドステージから生成されたバイナリと読み取り専用カタログをコピー
COPY --from=builder /app/server .
COPY --from=builder /app/catalog.db ./catalog.db

# 静的ファイルのディレクトリをコピー
COPY public/ ./public/

# ポート8080を開放
EXPOSE 8080

ENV CATALOG_DB=/app/catalog.db

# サーバーを実行
CMD ["./server"]
