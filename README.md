# manga

GitHub Pages: https://tomozo6.github.io/manga/

## ローカルで試す

Firebase Authentication はステージングの実プロジェクトを使用します。Firebase Console で
`localhost` が認可済みドメインになっていることを確認してください。

1. `application/.env.example` を参考に、`application/.env` を作成する。`ALLOWED_EMAILS`
   には自分の Google アカウントのメールアドレスを指定する。`go run .` はこのファイルを
   自動で読み込む（すでに設定されている環境変数は優先する）。
2. 環境変数を読み込んでサーバーを起動する。

   ```sh
   cd application
   gcloud auth application-default login
   gcloud auth application-default set-quota-project tomozo6
   go run .
   ```

3. スマートフォン、またはブラウザのモバイル表示で
   `http://localhost:8000` を開き、Google でログインする。

画像は非公開 GCS バケットのヒストリエを使います。認証済みの巻 API が返す GCS 署名 URL
だけで取得でき、URL は 1 時間で期限切れになります。

漫画カタログは作品ごとの YAML で編集し、SQLite を生成して利用します。YAML の形式と
ローカル・Dockerでの生成手順は [開発ドキュメント](docs/operations/catalog.md) を参照してください。
