# Mizumanju Server

Mizumanju は一定時間ごとに webcam の画像をメンバー間で共有するシステムです。
これはそのサーバです。

## Try Mizumanju

    $ docker run --name mizumanju-db -e MYSQL_DATABASE=mizumanju -e MYSQL_ROOT_PASSWORD=password -d mysql:5.6
    $ docker run --name mizumanju-ap --link mizumanju-db:db -d marcie001/mizumanju
    $ docker run --name mizumanju-web --link mizumanju-ap:ap -d -P marcie001/mizumanju-angular

ブラウザでアクセスしてください。ログイン画面が表示されるので、下のいずれかのユーザでログインしてください。

| username | password |
| -------- | -------- |
| admin    | password |
| user01   | password |
| user02   | password |

## DB Migrate

    $ go get github.com/marice001/mizumanju-server

MySQL のデータベースとユーザを作ってください。
開発環境は root でもいいでしょう。
ここでは docker で MySQL をインストールします。root のパスワードは `-e MYSQL_ROOT_PASSWORD=` で指定します。ここでは password としています。

    $ docker run -P -e MYSQL_ROOT_PASSWORD=password -d mysql:5.6
    $ mysqlport=$(docker inspect --format='{{range $p, $conf := .NetworkSettings.Ports}}{{(index $conf 0).HostPort}}{{end}}' $(docker ps -ql))
    $ mysql --host=127.0.0.1 --port=$mysqlport -u root -ppassword
    mysql> create database mizumanju character set utf8mb4 collate utf8mb4_unicode_ci;

Migration ツールである Goose をインストールします。
https://bitbucket.org/liamstask/goose

    $ go get bitbucket.org/liamstask/goose/cmd/goose

プロジェクトのルートディレクトリで次のコマンドを実行します。

    $ DATABASE_URL_GOOSE="tcp:127.0.0.1:$mysqlport*mizumanju/root/password" goose up

データベースにテーブルとデータが入っていることを確認してください。
