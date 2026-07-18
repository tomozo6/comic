package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// API用のレスポンス構造体
type MessageResponse struct {
	Message string `json:"message"`
}

func main() {
	mux := http.NewServeMux()

	// 1. 静的ファイルの配信 (例: localhost:8000/ で public ディレクトリを表示)
	// http.Dir(".") から見たパスを指定します
	fileServer := http.FileServer(http.Dir("./public"))
	mux.Handle("GET /", fileServer)

	// 2. APIの配信 (例: localhost:8000/api/hello)
	mux.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := MessageResponse{Message: "Hello from Go API!"}
		json.NewEncoder(w).Encode(response)
	})

	// サーバーの起動
	fmt.Println("Server running on http://localhost:8000")
	if err := http.ListenAndServe(":8000", mux); err != nil {
		panic(err)
	}
}
