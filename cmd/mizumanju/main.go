// パッケージ mizumanju は webcam の画像を一定時間ごとに共有するシステム。これはそのサーバの実装。
// 離れて共同作業するメンバーの現状を把握することで話しかけていいか、
// 離席中だからチャットの返事はないな、などのコミュニケーションの機微をつかむことを目的としている。
package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	_ "github.com/go-sql-driver/mysql"
	"github.com/marcie001/mizumanju"
)

// データベースへの接続、テンプレート準備、ルーティングの定義、サーバ起動を行う。
func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	h := flag.String("h", "", "Listen ip.")
	p := flag.Int("p", 8080, "Listen port.")
	d := flag.String("d", "root:password@tcp(127.0.0.1:3306)/mizumanju?charset=utf8mb4&parseTime=true&loc=Asia%2FTokyo", "DSN. See https://github.com/go-sql-driver/mysql#dsn-data-source-name")
	sh := flag.String("sh", "127.0.0.1", "SMTP host name.")
	sp := flag.Int("sp", 25, "SMTP port.")
	ss := flag.Bool("ss", false, "SMTP StartTLS support.")
	su := flag.String("su", "", "SMTP user name.")
	sw := flag.String("sw", "", "SMTP password.")
	n := flag.String("n", "mizumanju", "System name.")
	u := flag.String("u", "http://example.com/", "Base URL.")
	m := flag.String("m", "foo@example.com", "Mail adress of system.")
	pp := flag.Bool("pp", false, "Start debug server. See http://golang.org/pkg/net/http/pprof/")
	flag.Parse()

	if *pp {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	mizumanju.Start(*h, int32(*p), *d, *sh, *sp, *ss, *su, *sw, *n, *u, *m)
}
