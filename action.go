package mizumanju

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

// handleLogin は /api/login へのリクエストを処理する関数。
// 認証を行い、その結果を返す。認証 OK の場合、セッションに認証情報を格納する。
func login(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	log.Println("/login")
	param, ok := p.(*loginParams)
	if !ok {
		err = fmt.Errorf("Expected *loginParams, but actual is %T", p)
		log.Println(err)
		return
	}

	user, err := Authenticate(r, param.Username, param.Password)
	if err != nil {
		return
	}
	auth, _ := store.Get(r, sessionAuth)
	auth.Values["user"] = user
	auth.Save(r, w)
	b, err = json.Marshal(NewResponse(nil, &user))
	if err != nil {
		return
	}
	return
}

// handleGetUsers は /api/displaySettings へのリクエストを処理する関数。
// セッションの認証情報のユーザの表示設定を返す。
func getMyDisplaySettings(w http.ResponseWriter, r *http.Request, p params) ([]byte, error) {
	user, ok := context.Get(r, userkey).(*User)
	if !ok {
		return nil, errors.New("Server Error")
	}

	users, err := FindDisplaySettings(r, user.Id)
	if err != nil {
		return nil, err
	}

	var js []byte
	js, err = json.Marshal(NewResponse(nil, &users))
	if err != nil {
		return nil, err
	}
	return js, nil
}

// handleGetMe は /api/users/me へのリクエストを処理する関数。
// セッションの認証情報のユーザ情報を返す。
func getMe(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	user, ok := context.Get(r, userkey).(*User)
	if !ok {
		err = errors.New("Server Error")
		log.Println(err)
		return
	}

	u, err := FindUserById(r, user.Id)
	if err != nil {
		log.Println(err)
		return
	}
	b, err = json.Marshal(NewResponse(nil, &u))
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// handleSaveDisplay は /api/saveDisplay へのリクエストを処理する関数。
// ユーザの表示設定を保存する。
func postMyDisplaySettings(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	user, ok := context.Get(r, userkey).(*User)
	if !ok {
		err = errors.New("Server Error")
		log.Println(err)
		return
	}

	params, ok := p.(*[]User)
	if !ok {
		err = fmt.Errorf("Expected *[]User, but actual is %T", p)
		log.Println(err)
		return
	}

	err = UpsertDisplaySettings(r, user.Id, *params)
	if err != nil {
		log.Println(err)
		return
	}
	b, err = json.Marshal(NewResponse("OK", nil))
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// handleSaveImage は /api/saveImage へのリクエストを処理する関数。
// ユーザ画像を保存する。
func putMyImage(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	user, ok := context.Get(r, userkey).(*User)
	if !ok {
		err = errors.New("Server Error")
		log.Println(err)
		return
	}

	param, ok := p.(*imageParams)
	if !ok {
		err = fmt.Errorf("Expected *saveImageParams, but actual is %T", p)
		log.Println(err)
		return
	}

	err = SaveImage(r, user.Id, param.Image)
	if err != nil {
		return
	}

	b, err = json.Marshal(NewResponse("OK", nil))
	if err != nil {
		return
	}
	return b, nil
}

// handleServeImage は /api/users/{id:[0-9]+}/image へのリクエストを処理する関数。
// ユーザ画像を配信する。
func getUserImage(w http.ResponseWriter, r *http.Request, p params) ([]byte, error) {

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 32)
	if err != nil {
		log.Println(err)
		return nil, ErrBadRequest
	}

	img, err := GetImage(r, int32(id))
	if err == nil {
		w.Header().Set("Content-Type", "image/png")
	}
	return img, err
}

// handleDelUser は /api/users/{id:[0-9]+} への DELETE リクエストを処理する関数。
// ユーザを削除する。自分は削除できない
func deleteUser(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {

	vars := mux.Vars(r)
	id64, err := strconv.ParseInt(vars["id"], 10, 32)
	if err != nil {
		log.Println(err)
		return nil, ErrBadRequest
	}
	id32 := int32(id64)

	user, ok := context.Get(r, userkey).(*User)
	if !ok {
		return nil, errors.New("Server Error")
	}
	if user.Id == id32 {
		return nil, ErrBadRequest
	}

	err = DelUser(r, id32)
	if err != nil {
		log.Println(err)
		return
	}
	b, err = json.Marshal(NewResponse("OK", nil))
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// handleUsersGet は /api/users へのリクエストを処理する関数。
// 全ユーザを返す
func getUsers(w http.ResponseWriter, r *http.Request, p params) ([]byte, error) {
	users, err := FindAllUsers(r)
	if err != nil {
		return nil, err
	}

	var js []byte
	js, err = json.Marshal(NewResponse(nil, &users))
	if err != nil {
		return nil, err
	}
	return js, nil
}

// handleUsersPost は /api/users へのリクエストを処理する関数
// ユーザの作成を行う
func postUser(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	param, ok := p.(*User)
	if !ok {
		err = fmt.Errorf("Expected *User, but actual is %T", p)
		log.Println(err)
		return
	}

	var user User
	if param.Id > 0 {
		// editor の場合は本人のみ更新可能
		u, ok := context.Get(r, userkey).(*User)
		if !ok {
			return nil, errors.New("Server Error")
		}
		if u.Role == editor && param.Id != u.Id {
			err = ErrUnauthorized
			return
		}
		// editor の場合は Role は editor のみ
		if u.Role == editor {
			param.Role = editor
		}
		user, err = UpdateUser(r, *param)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		user, err = InsertUser(r, *param)
		if err != nil {
			log.Println(err)
			return
		}
	}
	b, err = json.Marshal(NewResponse(nil, &user))
	if err != nil {
		log.Println(err)
		return
	}
	return

}

// handleRecovery は /api/recovery/{key} へのリクエストを処理する関数
// ユーザのパスワード変更を行う
func recovery(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	param, ok := p.(*recoveryParams)
	if !ok {
		err = fmt.Errorf("Expected *recoveryParams, but actual is %T", p)
		log.Println(err)
		return
	}

	vars := mux.Vars(r)
	err = UpdatePasswordByRecoveryKey(r, vars["key"], param.Password)
	if err != nil {
		return
	}

	b, err = json.Marshal(NewResponse("Success to update password.", nil))
	if err != nil {
		return
	}
	return
}

// handleRecoveryRequest は /api/recovery へのリクエストを処理する関数
// パスワードリカバリキーの発行を行う
func requestRecovery(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	param, ok := p.(*recoveryRequestParams)
	if !ok {
		err = fmt.Errorf("Expected *recoveryRequestParams, but actual is %T", p)
		log.Println(err)
		return
	}

	err = CreateRecovery(r, param.Email)
	if err != nil {
		return
	}

	b, err = json.Marshal(NewResponse("Please check your mail box.", nil))
	if err != nil {
		return
	}
	return
}

// handlePassword は /api/updatePassword へのリクエストを処理する関数
// ユーザのパスワード変更を行う
func putMyPassword(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	u, ok := context.Get(r, userkey).(*User)
	if !ok {
		return nil, errors.New("Server Error")
	}

	param, ok := p.(*passwordParams)
	if !ok {
		err = fmt.Errorf("Expected *passwordParams, but actual is %T", p)
		log.Println(err)
		return
	}

	err = UpdatePasswordByAuthId(r, u.AuthId, param.CurrentPassword, param.NewPassword)
	if err != nil {
		return
	}

	b, err = json.Marshal(NewResponse("Success to update password.", nil))
	if err != nil {
		return
	}
	return
}

// getLicences は /api/licenses へのリクエストを処理する関数
func getLicenses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(NewResponse(nil, Licenses))
	if err != nil {
		return
	}
	w.Write(b)
}
