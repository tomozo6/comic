# 漫画カタログの管理

漫画の編集元は YAML、アプリケーションが読むデータは SQLite です。生成済みの
`catalog.db` はコミットしません。

## YAML の置き場所

作品ごとに `application/catalog/mangas/{manga_id}.yaml` を作成します。`id` は URL と
画像キーに使う不変の識別子です。`number` は画面に表示する巻番号で、変更しても構いません。

```yaml
id: demo-comic
title: ふしぎな青い本
author_name: 開発用サンプル
cover_object_key: manga/demo-comic/cover.webp # 任意
volumes:
  - id: volume-1
    number: 1
    title: 青い本のひみつ
    page_count: 3
    page_extension: webp
```

表紙を指定しない場合は `cover_object_key` を省略します。同じ作品内では `volumes.id` と
`number` の重複を禁止します。`page_count` は 1 以上、`page_extension` は英数字だけです。

各ページを YAML に列挙しません。リーダーは次の規則でページ画像キーを作ります。

```text
manga/{manga_id}/{volume_id}/{page:03d}.{page_extension}
```

たとえば上記の1ページ目は `manga/demo-comic/volume-1/001.webp` です。同一巻内の画像は
同じ拡張子にそろえます。

## ローカル開発

通常どおりアプリケーションを起動します。

```sh
cd application
set -a; source .env; set +a
go run .
```

`CATALOG_DB` を指定しない限り、起動時に `catalog/mangas/*.yaml` から一時 SQLite を生成します。
生成したファイルはプロセス終了時に削除されます。YAML の場所を変える必要がある場合だけ、
`CATALOG_SOURCE_DIR` を指定します。

SQLite の内容だけを確認したい場合は、次のコマンドで任意の出力先へ生成できます。

```sh
cd application
go run ./cmd/catalog-build -source catalog/mangas -output /tmp/catalog.db
```

## Docker / 本番

Dockerfile はビルドステージで同じ `catalog-build` を実行します。生成された `catalog.db` だけを
実行イメージにコピーし、`CATALOG_DB=/app/catalog.db` を設定します。実行中のアプリケーションは
このDBを読み取り専用で開き、YAML を読み込んだりDBを書き換えたりしません。

カタログを変更したら、YAML をレビューしてイメージを再ビルド・再デプロイします。
