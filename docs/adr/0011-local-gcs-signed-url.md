# ADR 0011: ローカル開発では IAM による GCS 署名付き URL を発行する

- 状態: 採用
- 日付: 2026-07-20

## 背景

実際の漫画画像を、非公開バケット `tomozo-manga-images` に配置した。ローカルでの
アプリケーション実装でも、ブラウザが実画像を読む経路を確認したい。一方、サービス
アカウント秘密鍵を開発端末に保存してはならない。

## 決定

- ローカルの Go アプリケーションは Application Default Credentials（ADC）を使う。
  開発者は `gcloud auth application-default login` で ADC を設定する。
- URL 署名専用のサービスアカウント `manga-media-signer` を Terraform で作成する。
  このサービスアカウントには `tomozo-manga-images` バケットのオブジェクト閲覧権限だけを
  付与する。
- ローカル開発者には、そのサービスアカウントに対する
  `roles/iam.serviceAccountTokenCreator` を付与する。アプリケーションは ADC で認証した
  開発者権限を使い、IAM Credentials API の `signBlob` により V4 GET 署名 URL を発行する。
- サービスアカウント秘密鍵は作成・配布しない。
- 署名付き URL の有効期限は 1 時間とする。215 ページ程度の巻を読む間に、未読込の画像が
  失効することを避ける。
- アプリケーションはすべての環境で GCS 署名 URL を発行する。ローカル配信 URL や埋込み画像は
  持たない。

## 影響

- ローカル環境では ADC の設定と、開発者への Token Creator 権限が必須になる。
- 署名付き URL は有効期限中、Firebase 認証なしでアクセスできる bearer URL である。
  URL は API 応答以外へ記録・共有しない。
- Cloud Run では、割り当てたサービスアカウントを ADC が自動検出する。そのサービスアカウント
  には `manga-media-signer` に対する Token Creator 権限が必要である。
