# GCS 漫画画像の運用

本番の漫画画像は、GCP プロジェクト `tomozo6` の非公開バケット
`gs://tomozo-manga-images` に保存する。バケットの公開設定とアクセス方針は
[ADR 0010](../adr/0010-private-gcs-manga-images.md) を参照する。

## オブジェクトキーの構成

GCS に実ディレクトリはなく、`/` を含むオブジェクトキーをコンソールが
ディレクトリのように表示する。漫画画像は次のプレフィックス構成にする。

```text
gs://tomozo-manga-images/
└── manga/
    └── {manga_id}/
        ├── cover.{extension}                 # 任意: 作品表紙
        └── {volume_id}/
            ├── 001.{page_extension}          # 必須: 1 ページ目
            ├── 002.{page_extension}
            └── ...
```

各ページのキーは、カタログの `id`、`volumes.id`、`page_extension` から次の規則で
決まる。ページ番号は 3 桁のゼロ埋めとする。

```text
manga/{manga_id}/{volume_id}/{page:03d}.{page_extension}
```

例: `manga/historie/001/001.jpeg`

`manga_id` と `volume_id` は URL とオブジェクトキーに使う不変の識別子である。
表示用の巻番号（`number`）を変更しても、GCS 上のパスは変更しない。

作品表紙は `cover_object_key` に任意のキーを指定できる。上の `cover.{extension}` は
推奨の配置であり、必須ではない。巻ごとの表紙を設定する場合も、カタログの
`cover_object_key` と同じキーでオブジェクトを配置する。

## カタログとの対応

画像を投入する前に、`application/catalog/mangas/{manga_id}.yaml` の `page_count` と
`page_extension` を確認する。`page_count` の 1 から指定値まで、欠番なく同じ拡張子の
ファイルを配置する。カタログの編集方法と検証方法は
[漫画カタログの管理](catalog.md) を参照する。

## 投入と確認

ローカルの画像ディレクトリを投入する場合の例は次のとおり。

```sh
gcloud storage ls gs://tomozo-manga-images/manga/historie/001/
```

初回投入や差し替えでは、`gcloud storage ls` でキー、連番、拡張子がカタログと一致する
ことを確認してからデプロイする。バケットはバージョニングが有効なので、同一キーを
上書きした場合も旧世代は保持される。

実際の漫画画像をこのリポジトリや公開される Docsify 文書に追加してはならない。

## ローカルで実画像を読む

初回だけ、Terraform にローカル開発者を署名権限の対象として渡して適用する。

```sh
cd terraform
terraform apply -var='local_media_signer_member=user:your-account@example.com'
```

サービスアカウント秘密鍵は作成しない。ローカル端末では ADC を設定してから、GCS 署名 URL
署名 URL を発行してアプリケーションを起動する。

```sh
gcloud auth application-default login
gcloud auth application-default set-quota-project tomozo6
cd application
set -a; source .env; set +a
go run .
```

GCS 署名 URL は 1 時間有効であり、URL 自体をログや文書に記録してはならない。
