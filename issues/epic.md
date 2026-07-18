 # 漫画閲覧サイト v1 アーキテクチャ案

  ## Summary

  - v1 は Cloud Run 1つに HTML 配信/API を同居させる構成にする。
  - 認証は Firebase Auth の Google ログインのみ。
  - 漫画画像は private GCS bucket に置き、Cloud Run が Firebase ID token を検証して 短命の GCS 署名付きURLを発行する。
  - 閲覧権限は v1 では「ログイン済みユーザー全員が全作品を閲覧可能」とする。

  ## Key Architecture

  - Browser
      - Cloud Run から index.html / JS / CSS を取得
      - Firebase Auth SDK で Google ログイン
      - API 呼び出し時に Authorization: Bearer <Firebase ID token> を付ける
      - API から受け取った短命署名URLで GCS 画像を直接表示

  - Cloud Run Go App
      - 静的ファイル配信: /, /assets/*
      - API:
          - GET /api/me: Firebase ID token 検証確認
          - GET /api/manga: 作品一覧を返す
          - GET /api/manga/{mangaId}/chapters/{chapterId}: ページ一覧と署名付きURLを返す

      - Firebase Admin SDK または Google identitytoolkit 相当で ID token を検証
      - GCS object に対して有効期限 5〜15分程度の署名付きURLを発行
      - カタログメタデータはコンテナイメージに同梱した SQLite を読み取り専用で参照する

  - SQLite catalog
      - 管理対象は作品、作者、巻、章、ページ、および GCS object key とする。作品一覧とページ一覧 API はこのカタログを参照する
      - スキーマ定義 (DDL)、初期データ (DML)、および必要な画像配置との整合性検証をリポジトリで管理する
      - コンテナビルド時に空の SQLite DB へ DDL/DML を適用し、生成した catalog.db を runtime image に COPY する
      - Cloud Run のアプリケーションは catalog.db を更新しない。複数インスタンス・複数リビジョンがそれぞれ同一の読み取り専用コピーを参照してよい
      - データ更新時は DDL/DML を変更して新しいイメージをビルド・デプロイする。稼働中の Cloud Run ローカルファイルシステムを更新対象にしない
      - SQLite ファイルは Cloud Run の永続ストレージではない。オンライン更新、ユーザーごとの状態、または複数サービスからの書き込みが必要になった時点で Cloud SQL などの永続 DB へ移行する

  - GCS
      - bucket は public access 無効
      - 画像配置例:
          - manga/{mangaId}/{chapterId}/{pageNo}.jpg

      - Cloud Run runtime service account のみに object read 権限を付与

  - Terraform
      - 追加する主なリソース:
          - Artifact Registry
          - Cloud Run service
          - Cloud Run service account
          - GCS bucket
          - IAM: Cloud Run service account に GCS read 権限
          - Secret Manager: OAuth client secret など秘匿値

      - 既存の Firebase Auth / Identity Platform 設定は継続利用
      - 現在 locals.tf にある OAuth client secret は Secret Manager または terraform.tfvars 管理へ移す

  ## Implementation Notes

  - Cloud Run 1つ構成で始めるのが妥当。理由は、HTML/API/認証検証/署名URL発行を一箇所で管理でき、v1 の開発速度が高いから。
  - 画像そのものは Cloud Run を経由させず、署名付きURLで GCS から直接返す。Cloud Run の帯域・CPU・レスポンス時間を画像配信で消費しない。
  - 署名付きURLの有効期限はまず 10分にする。
  - API は Firebase ID token が無い、期限切れ、不正な場合は 401 を返す。
  - v1 のカタログは、更新のない読み取り専用データであるため SQLite をコンテナイメージへ同梱する。GCS の object key を DB に持たせ、画像配信時に
    動的に署名付きURLへ変換する。
  - DB 生成は Dockerfile の runtime stage ではなく build stage で完結させる。完成した catalog.db とアプリケーションだけを runtime image に含める。
  - Go の SQLite driver は CGO 要否を確認する。CGO を使わない現行ビルドを維持する場合は pure-Go driver を採用する。
  - 作品単位の購入/権限管理、オンラインでのカタログ更新、またはユーザー状態が必要になった段階で Firestore か Cloud SQL を追加する。

  ## Test Plan

  - ローカル:
      - index.html で Google ログインできること
      - Firebase ID token を Cloud Run API に送れること
      - 未ログイン時に API が 401 を返すこと
      - DDL/DML から catalog.db を再生成できること
      - catalog.db の作品・巻・章・ページと GCS object key の対応が検証できること

  - Cloud Run:
      - / でHTMLが返ること
      - /api/me がログイン済みユーザー情報を返すこと
      - /api/manga/... が署名付きURLを返すこと
      - 返された署名付きURLで private GCS 画像を表示できること
      - URL期限切れ後に画像へアクセスできないこと
      - 複数インスタンスで同じカタログを返せること

  - Terraform:
      - terraform plan が安定すること
      - GCS bucket が public でないこと
      - Cloud Run service account の権限が GCS read に限定されていること

  ## Assumptions

  - v1 はシンプル優先。
  - 閲覧可能範囲はログイン済みユーザー全員。
  - 画像配信は短命 GCS 署名付きURL。
  - カタログのオンライン更新は行わない。カタログ変更はイメージの再ビルドと Cloud Run の新リビジョンのデプロイで反映する。
  - SQLite はコンテナに同梱する読み取り専用データであり、Cloud Run のローカルディスクを永続ストレージとして扱わない。
  - 独自ドメイン stg-comic.tomozo6.com は将来的に Cloud Run に向ける。
  - 高トラフィック化したら、HTML 配信を Firebase Hosting/Cloud CDN に分離し、画像配信も Cloud CDN 署名URLへ移行する。
