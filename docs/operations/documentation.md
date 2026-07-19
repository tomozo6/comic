# 開発ドキュメントの運用

このディレクトリはDocsifyで表示する、開発・運用者向けの設計書である。GitHub Pagesでは
`docs/` を公開元にする。

## Docsifyの構成

- `index.html` はDocsifyの起点で、`README.md` をホームとして読み込む。
- `_sidebar.md` は全ページ共通のナビゲーションである。
- `.nojekyll` はGitHub PagesのJekyll処理を止める。これにより、先頭が `_` の
  `_sidebar.md` が公開対象から除外されない。

`_sidebar.md` はDocsifyの正しい予約ファイル名であるため、別名へ変更しない。

## 文書の置き場所

| 内容 | 場所 |
| --- | --- |
| プロダクトの導線・利用者要件 | `product/` |
| ワイヤーフレームなどの設計資料 | `design/` |
| システム構成・実装方針 | `architecture/` |
| ローカル・デプロイなどの手順 | `operations/` |
| 意思決定の記録 | `adr/` |
| 共通用語 | `glossary.md` |

文書を追加・移動したら、必ず `_sidebar.md` とホームの `README.md` へ導線を追加する。

## 公開前の確認

GitHub Pagesは公開インターネット上で閲覧できる。秘密情報、個人情報、実際の漫画画像、
サービスアカウント鍵を `docs/` に含めない。
