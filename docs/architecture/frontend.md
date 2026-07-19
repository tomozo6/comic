# フロントエンド構成

実行時のフロントエンドは、Go アプリケーションが `application/public/` から配信する静的な
複数ページアプリケーションである。ビルドツールやJavaScriptフレームワークは使わず、ブラウザの
ネイティブ ES Modules を使う。

## ディレクトリ

```text
application/public/
├── index.html                 # ログイン
├── library.html               # 作品一覧
├── manga.html                 # 巻一覧
├── reader.html                # 漫画リーダー
└── assets/
    ├── css/site.css           # 共通スタイル
    └── js/
        ├── firebase.js        # Firebase初期化だけを担当
        ├── auth.js            # 認証確認とログアウト
        ├── api.js             # 認証済みAPI呼び出し
        ├── routes.js          # URLからIDを取得
        ├── ui.js              # エラー表示などの小さなUI操作
        └── pages/             # 画面ごとの読み込み・描画
```

HTMLは画面の構造と対応する `pages/` モジュールの読み込みだけを持つ。共有ヘッダーのHTMLは小さいため
各画面へ明示的に記述し、JavaScriptで動的に組み立てない。共通化するのは振る舞いだけである。

## 変更時の指針

- 認証の振る舞いは `auth.js`、HTTP通信は `api.js` に置く。
- 画面固有のDOM生成・表示処理は対応する `pages/{画面名}.js` に置く。
- APIから受け取った文字列を `innerHTML` へ渡さず、`textContent` とDOM APIで表示する。
- 設計用ワイヤーフレームは実行用資産に含めず、[画面設計](../design/wireframes/README.md)で管理する。
