# ==========================================
# 1. ビルド用のステージ (Build Stage)
# ==========================================
FROM golang:1.22-alpine AS builder

# ワークディレクトリの設定
WORKDIR /app

# モジュール管理ファイルがある場合はコピーしてダウンロード
# (今回はシンプルな構成なので、なければスルーされます)
COPY go.mod* go.sum* ./
RUN if [ -f go.mod ]; then go mod download; fi

# ソースコードをコピー
COPY main.go .

# Goバイナリをビルド (CGOを無効化し、Linux向けに最適化)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server main.go

# ==========================================
# 2. 実行用のステージ (Run Stage)
# ==========================================
FROM alpine:latest

WORKDIR /app

# ビルドステージから生成されたバイナリだけをコピー
COPY --from=builder /app/server .

# 静的ファイルのディレクトリをコピー
COPY public/ ./public/

# ポート8080を開放
EXPOSE 8080

# サーバーを実行
CMD ["./server"]
