// パッケージ mizumanju は web server が行う処理。
package mizumanju

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/mail"
	"net/url"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var (
	// データベースインスタンス
	db *sql.DB
	// セッションストア
	store = sessions.NewCookieStore(securecookie.GenerateRandomKey(64))
	// テンプレート
	tmpl *template.Template
	// SMTP 設定
	smtpConf   *SmtpConf
	systemConf *SystemConf
	// ErrBadRequest は HTTP Status Code 401 に相応しいエラー
	ErrBadRequest error = errors.New("Bad Request.")
	// ライセンス情報
	Licenses = []License{
		License{
			Title:      "User shape",
			Source:     "http://www.flaticon.com/free-icon/user-shape_25634",
			Author:     "Dave Gandy",
			AuthorURL:  "http://www.flaticon.com/authors/dave-gandy",
			License:    "CC BY 3.0",
			LicenseURL: "http://creativecommons.org/licenses/by/3.0/",
		},
	}
)

const (
	// セッション内の認証情報のキー
	sessionAuth = "SessionAuth"
	// context 内のログインユーザのキー
	userkey = "LoginUser"
	// エラー時のレスポンステンプレート
	errJsTmpl = `{"msgs":{"global":["%s"]},"data":null}`
	// 404 エラー時のレスポンス
	err404Tmpl = `{"msgs":{"global":["404 page not found"]},"data":null}`
	// ロール admin
	admin = "admin"
	// ロール editor
	editor = "editor"
	// ユーザに表示する全体的なメッセージであることを表すキー
	GlobalMsg = "global"
)

// starg はデータベースへの接続、テンプレート準備、ルーティングの定義、サーバ起動を行う。
func Start(host string, port int32, dsn string, smtpHost string, smtpPort int, startTls bool, smtpUserName string, smtpPassword string, systemName string, systemUrl string, systemMailAddress string) {

	baseUrl, err := url.Parse(systemUrl)
	if err != nil {
		log.Fatal(err)
	}
	systemConf = &SystemConf{
		Name: systemName,
		URL:  baseUrl,
		Mail: &mail.Address{
			Name:    systemName,
			Address: systemMailAddress,
		},
	}

	smtpConf = &SmtpConf{
		Host:     smtpHost,
		Port:     smtpPort,
		User:     smtpUserName,
		Password: smtpPassword,
		Sender:   systemMailAddress,
		TLS:      startTls,
	}
	tmpl = CreateTemplate()

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if derr := db.Close(); derr != nil {
			log.Fatal(derr)
		}
	}()

	gob.Register(&User{})

	router := mux.NewRouter()

	router.HandleFunc("/api/login", makeCtxHandler(makeOne(validateLogin, login), new(loginParams))).Methods("POST")
	router.HandleFunc("/api/users/me/displaySettings", makeCtxHandler(makeAuthedAction(getMyDisplaySettings), nil)).Methods("GET")
	users := make([]User, 0, 32)
	router.HandleFunc("/api/users/me/displaySettings", makeCtxHandler(makeAuthedAction(postMyDisplaySettings), &users)).Methods("POST")
	router.HandleFunc("/api/users/me/image", makeCtxHandler(makeAuthedAction(putMyImage), new(imageParams))).Methods("PUT")
	router.HandleFunc("/api/users/{id:[0-9]+}/image", makeCtxHandler(makeAuthedAction(getUserImage), nil)).Methods("GET")
	router.HandleFunc("/api/users/me/password", makeCtxHandler(makeAuthedAction(makeOne(validatePassword, putMyPassword)), new(passwordParams))).Methods("PUT")
	router.HandleFunc("/api/users/{id:[0-9]+}", makeCtxHandler(makeAuthedAction(deleteUser, admin), nil)).Methods("DELETE")
	router.HandleFunc("/api/users/me", makeCtxHandler(makeAuthedAction(getMe), nil)).Methods("GET")
	router.HandleFunc("/api/users", makeCtxHandler(makeAuthedAction(getUsers, admin), nil)).Methods("GET")
	router.HandleFunc("/api/users", makeCtxHandler(makeAuthedAction(makeOne(validateUser, postUser), admin), new(User))).Methods("POST")
	router.HandleFunc("/api/users", makeCtxHandler(makeAuthedAction(makeOne(validateUser, postUser)), new(User))).Methods("PUT")
	router.HandleFunc("/api/recovery/{key:[a-z0-9\\-]+}", makeCtxHandler(makeOne(validateRecovery, recovery), new(recoveryParams))).Methods("PUT")
	router.HandleFunc("/api/recovery", makeCtxHandler(makeOne(validateRecoveryRequest, requestRecovery), new(recoveryRequestParams))).Methods("POST")
	router.HandleFunc("/api/licenses", getLicenses).Methods("GET")
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil))
}

// params はリクエストボディの JSON 形式パラメタを表す
type params interface{}

// actionFunc は各パスに対する個別処理を行う関数の型。
// []byte はレスポンスボディとしてクライアントに送信される。
// また、error が nil でないとき、エラー用ステータスコードとメッセージがクライアントに送信される。
type actionFunc func(w http.ResponseWriter, r *http.Request, p params) ([]byte, error)

// makeAuthedAction は fn に事前認証チェック機能を付加する関数。
// fn の処理の前に認証チェックを行い、ログインユーザ情報を context に格納する
func makeAuthedAction(fn actionFunc, roles ...string) actionFunc {
	return func(w http.ResponseWriter, r *http.Request, p params) ([]byte, error) {
		auth, _ := store.Get(r, sessionAuth)
		user, ok := auth.Values["user"].(*User)
		if !ok {
			return nil, ErrUnauthorized
		}
		if roles != nil && !inArray(roles, user.Role) {
			log.Printf("Unauthorized. ID: %d, Name: %s", user.Id, user.Name)
			return nil, ErrUnauthorized
		}
		context.Set(r, userkey, user)
		return fn(w, r, p)
	}
}

// inArray は a に s と同等の要素が格納されているとき true を返す関数。
func inArray(a []string, s string) bool {
	for _, e := range a {
		if e == s {
			return true
		}
	}
	return false

}

// makeCtxHandler は fn 処理の前に共通事前処理と後処理を付加する関数。
// 事前処理は DB インスタンスを context にセットする。
// 後処理はレスポンスの書き出し。
func makeCtxHandler(fn actionFunc, p params) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SetDB(r, db)
		SetMailTmpl(r, tmpl)
		SetSmtpConf(r, smtpConf)
		SetSystemConf(r, systemConf)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		var err error
		if p != nil {
			err = json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, fmt.Sprintf(errJsTmpl, err.Error()))
				return
			}
		}

		ret, err := fn(w, r, p)
		switch {
		case err == ErrNotFound, err == sql.ErrNoRows:
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, err404Tmpl)
			return
		case err == ErrUnauthorized:
			log.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, fmt.Sprintf(errJsTmpl, err.Error()))
			return
		case err == ErrBadRequest:
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, fmt.Sprintf(errJsTmpl, err.Error()))
			return
		case err == ErrValidation:
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ret)
			return
		case err != nil:
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, fmt.Sprintf(errJsTmpl, err.Error()))
			return
		}
		w.Write(ret)
	}
}

// makeOne は複数の actionFunc を一つの actionFunc にする関数。
// actionFunc の戻り値のどちらか一方が nil でない場合、actionFunc の実行を中断する
func makeOne(fns ...actionFunc) actionFunc {
	return func(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
		for _, fn := range fns {
			b, err = fn(w, r, p)
			if b != nil || err != nil {
				return
			}
		}
		return
	}
}

// Response はレスポンス用の構造体
type Response struct {
	Msgs map[string][]string `json:"msgs"`
	Data interface{}         `json:"data"`
}

// NewResponse は Response 構造体を生成する関数
func NewResponse(msgs interface{}, data interface{}) *Response {
	var m map[string][]string
	switch msgs := msgs.(type) {
	case string:
		m = make(map[string][]string)
		a := make([]string, 1, 1)
		a[0] = msgs
		m[GlobalMsg] = a
	case []string:
		m = make(map[string][]string)
		m[GlobalMsg] = msgs
	case map[string][]string:
		m = msgs
	default:
		m = make(map[string][]string)
	}

	return &Response{
		Msgs: m,
		Data: data,
	}
}

// License はライセンス情報を表す構造体
type License struct {
	Title      string `json:"title"`
	Source     string `json:"source"`
	Author     string `json:"author"`
	AuthorURL  string `json:"authorURL"`
	License    string `json:"license"`
	LicenseURL string `json:"licenseURL"`
}
