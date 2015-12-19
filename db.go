package mizumanju

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"github.com/marcie001/mizumanju/imgmap"
)

var (
	// ErrNotFound はデータがないことを表すエラー
	ErrNotFound error = errors.New("Data not found")
	// ErrUnauthorized は認証されていないことを表すエラー
	ErrUnauthorized error = errors.New("Unauthorized")
)

type User struct {
	Id          int32     `json:"id"`
	Name        string    `json:"name"`
	VoiceChatID string    `json:"voiceChatId"`
	Hide        bool      `json:"hide"`
	Image       string    `json:"image"`
	Role        string    `json:"role"`
	AuthId      string    `json:"authId"`
	DeleteFlag  bool      `json:"deleteFlag"`
	Email       string    `json:"email"`
	OrderNo     int32     `json:"orderNo"`
	Created     time.Time `json:"created"`
}

type UserStatus struct {
	UserId  int32     `json:"userId"`
	Status  string    `json:"status"`
	Updated time.Time `json:"updated"`
}

// context に登録するキー
type key int32

const (
	// context に登録する DB のキー
	dbkey key = 1
	// context に登録する SmtpConf のキー
	smtpkey key = 2
	// context に登録する Template のキー
	tmplkey key = 3
	// context に登録する SystemConf のキー
	systemkey key = 4
	// 認証時 SQL
	sqlFindByAuthId string = "SELECT id, auth_id, name, voice_chat_id, role, password, email, created FROM users WHERE auth_id = ? AND delete_flag = false"
	// Email でユーザを検索
	sqlFindByEmail string = "SELECT id, name, voice_chat_id, role, password, email, created FROM users WHERE email = ? AND delete_flag = false"
	// ユーザ取得
	sqlFindById string = "SELECT id, name, voice_chat_id, role, auth_id, email, created FROM users WHERE id = ? AND delete_flag = false"
	// ユーザステータス取得
	sqlFindUserStatusByUserId string = "SELECT user_id, status, updated FROM user_status WHERE user_id = ?"
	// 表示設定取得 SQL
	sqlFindDisplay string = "SELECT u.id, u.name, u.voice_chat_id, CASE WHEN uds.hide IS NULL THEN false ELSE uds.hide END, CASE WHEN uds.order_no IS NULL THEN -1 ELSE uds.order_no END FROM users u LEFT OUTER JOIN user_display_settings uds ON u.id = uds.target_user_id AND uds.user_id = ? WHERE u.id <> ? AND u.delete_flag = false ORDER BY uds.order_no, u.id DESC"
	// 表示設定登録/更新 SQL
	sqlUpsertDisplay string = "INSERT INTO user_display_settings (order_no, hide, user_id, target_user_id) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE order_no = ?, hide = ?"
	// ユーザ登録 SQL
	sqlInsertUser string = "INSERT INTO users (auth_id, name, voice_chat_id, role, password, email, created) VALUES (?, ?, ?, ?, '', ?, ?)"
	// ユーザ登録 SQL パスワードリカバリ
	sqlInsertUserPasswdRecovery string = "INSERT INTO user_password_recovery (id, user_id, created) VALUES (?, ?, ?)"
	// パスワードリカバリ情報の取得
	sqlFindUserPasswdRecovery string = "SELECT upr.user_id, upr.created, u.created FROM user_password_recovery upr INNER JOIN users u ON upr.user_id = u.id WHERE upr.id = ?"
	// パスワードリカバリ情報の削除
	sqlDeleteUserPasswdRecovery string = "DELETE FROM user_password_recovery WHERE id = ?"
	// ユーザ更新 SQL
	sqlUpdateUser string = "UPDATE users SET name = ?, voice_chat_id = ?, role = ?, email = ?, delete_flag = ? WHERE id = ?"
	// ユーザステータス更新 SQL
	sqlUpdateUserStatus string = "INSERT INTO user_status (user_id, status, updated) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE status = ?, updated =?"
	// ユーザ削除 SQL
	sqlDeleteUser string = "UPDATE users SET delete_flag = true, password = '', email = '' WHERE id = ?"
	// パスワード変更 SQL
	sqlUpdatePasswd string = "UPDATE users SET password = ? WHERE id = ? AND delete_flag = false"
	// 全ユーザ取得
	sqlFindAllUsers string = "SELECT id, auth_id, name, voice_chat_id, role, email, delete_flag FROM users ORDER BY id"
)

// SetDB は DB インスタンスを context に保存する関数。
func SetDB(r *http.Request, db *sql.DB) {
	context.Set(r, dbkey, db)
}

// SetSystemName はシステム名を context に保存する関数。
func SetSystemConf(r *http.Request, cnf *SystemConf) {
	context.Set(r, systemkey, cnf)
}

// Authenticate はデータベースに問い合わせ認証を行う関数。
// 認証成功時、該当ユーザの情報を返す。
func Authenticate(r *http.Request, inId, inPasswd string) (User, error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		return User{}, errors.New("DB instance not found.")
	}
	var (
		id                                        int32
		authId, name, vcid, role, password, email string
		created                                   time.Time
	)
	err := db.QueryRow(sqlFindByAuthId, inId).Scan(&id, &authId, &name, &vcid, &role, &password, &email, &created)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("AuthId: %s", inId)
		return User{}, ErrUnauthorized
	case err != nil:
		return User{}, err
	default:
		// パスワードチェック
		chk, err := checkPassword(inPasswd, password, created)
		if err != nil {
			return User{}, err
		}
		if !chk {
			return User{}, ErrUnauthorized
		}
		return User{
			Id:          id,
			AuthId:      authId,
			Name:        name,
			VoiceChatID: vcid,
			Image:       fmt.Sprint("/api/users/", id, "/image"),
			Role:        role,
			Created:     created,
		}, nil
	}
}

// checkPassword はパスワードが正しいかチェックする関数。
func checkPassword(userinput, password string, created time.Time) (bool, error) {
	hashed, err := hashPassword(userinput, created)
	if err != nil {
		return false, err
	}
	return password == hashed, nil
}

// FindDisplaySettings は userId のユーザ表示設定を取得する関数。
func FindDisplaySettings(r *http.Request, userId int32) (users []User, err error) {
	users = make([]User, 0, 32)

	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		return
	}
	var rows *sql.Rows
	rows, err = db.Query(sqlFindDisplay, userId, userId)
	if err != nil {
		return
	}
	defer func() {
		if rerr := rows.Close(); err == nil {
			err = rerr
		}
	}()
	for rows.Next() {
		var (
			id, order  int32
			name, vcid string
			hide       bool
		)
		err = rows.Scan(&id, &name, &vcid, &hide, &order)
		if err != nil {
			return
		}
		users = append(users, User{
			Id:          id,
			Name:        name,
			VoiceChatID: vcid,
			Hide:        hide,
			Image:       fmt.Sprint("/api/users/", id, "/image"),
			OrderNo:     order,
		})
	}
	return
}

// UpsertDisplaySettings は複数表示設定を更新または挿入する関数
func UpsertDisplaySettings(r *http.Request, userId int32, users []User) (err error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		log.Println(err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	for _, u := range users {
		if err = UpsertDisplaySetting(r, tx, userId, u.Id, u.OrderNo, u.Hide); err != nil {
			log.Println(err)
			return
		}
	}
	return
}

// FindUserById は id でユーザ情報を取得する関数
func FindUserById(r *http.Request, id int32) (u User, err error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		log.Println(err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	u, err = findUserById(r, tx, id)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// findUserById は id でユーザ情報を取得する関数
func findUserById(r *http.Request, tx *sql.Tx, id int32) (u User, err error) {
	err = tx.QueryRow(sqlFindById, id).Scan(&u.Id, &u.Name, &u.VoiceChatID, &u.Role, &u.AuthId, &u.Email, &u.Created)
	return
}

// FindUserStatusByUserId は userId でユーザステータスを取得する関数
func FindUserStatusByUserId(r *http.Request, userId int32) (u UserStatus, err error) {

	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		return
	}

	err = db.QueryRow(sqlFindUserStatusByUserId, userId).Scan(&u.UserId, &u.Status, &u.Updated)
	if err != nil {
		return
	}
	return
}

// UpsertDisplaySetting は表示設定を更新または挿入する関数
func UpsertDisplaySetting(r *http.Request, tx *sql.Tx, userId int32, targetUserId int32, orderNo int32, hide bool) (err error) {
	_, err = findUserById(r, tx, userId)
	if err == sql.ErrNoRows {
		log.Printf("ignored: user id(%d)", userId)
		return nil
	} else if err != nil {
		log.Println(err)
		return
	}

	_, err = tx.Exec(sqlUpsertDisplay, orderNo, hide, userId, targetUserId, orderNo, hide)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// イメージマップのインスタンス
var images = imgmap.New()

// SaveImage はユーザ画像をメモリ上に保存する関数。
func SaveImage(r *http.Request, userId int32, image string) error {
	b, err := base64.StdEncoding.DecodeString(image)
	if err != nil {
		return err
	}

	images.Set(userId, b)

	return nil
}

// GetImage はユーザ画像をメモリ上から取得する関数。
func GetImage(r *http.Request, userId int32) ([]byte, error) {
	return images.Get(userId)
}

// hashPassword はパスワードをハッシュする関数。
func hashPassword(password string, created time.Time) (string, error) {
	buf := new(bytes.Buffer)
	ms := created.UnixNano()
	err := binary.Write(buf, binary.LittleEndian, ms)
	if err != nil {
		return "", err
	}
	b := []byte(password)
	var h [64]byte
	for i := 0; i < 50; i++ {
		if i == 0 {
			b = append(b, buf.Bytes()...)
		} else {
			b = append(h[:], buf.Bytes()...)
		}
		h = sha512.Sum512(b)
	}
	return fmt.Sprintf("%x", h), nil
}

// FindAllUsers は全ユーザをデータベースから取得する関数
func FindAllUsers(r *http.Request) (users []User, err error) {
	users = make([]User, 0, 32)

	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		return
	}

	var rows *sql.Rows
	rows, err = db.Query(sqlFindAllUsers)
	if err != nil {
		return
	}
	defer func() {
		if rerr := rows.Close(); err == nil {
			err = rerr
		}
	}()
	for rows.Next() {
		var (
			id                              int32
			authId, name, vcid, role, email string
			delFlg                          bool
		)
		err = rows.Scan(&id, &authId, &name, &vcid, &role, &email, &delFlg)
		if err != nil {
			return
		}
		users = append(users, User{
			Id:          id,
			AuthId:      authId,
			Name:        name,
			VoiceChatID: vcid,
			Role:        role,
			DeleteFlag:  delFlg,
			Email:       email,
		})
	}
	return
}

// InsertUser はユーザ情報をデータベースに挿入し、メールで通知する関数
func InsertUser(r *http.Request, user User) (u User, err error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		log.Println(err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	rslt, err := tx.Exec(sqlInsertUser, user.AuthId, user.Name, user.VoiceChatID, user.Role, user.Email, time.Now())
	if err != nil {
		log.Println(err)
		return
	}
	id, err := rslt.LastInsertId()
	if err != nil {
		log.Println(err)
		return
	}
	user.Id = int32(id)

	key, err := createRecoveryKey(r, tx, user.Id)
	if err != nil {
		log.Println(err)
		return
	}

	err = SendInvitation(r, user.Name, user.Email, key)
	if err != nil {
		log.Println(err)
		return
	}

	return user, nil
}

// createRecoveryKey はパスワードリカバリキーを生成し、データベースに保存する関数
func createRecoveryKey(r *http.Request, tx *sql.Tx, userId int32) (key string, err error) {
	key, err = uuid()
	if err != nil {
		return
	}
	_, err = tx.Exec(sqlInsertUserPasswdRecovery, key, userId, time.Now())
	return
}

// CreateRecovery はパスワードリカバリキーを生成し、メールでユーザに通知する関数
func CreateRecovery(r *http.Request, email string) (err error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		log.Println(err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var (
		id                         int32
		name, vcid, role, password string
		created                    time.Time
	)
	err = db.QueryRow(sqlFindByEmail, email).Scan(&id, &name, &vcid, &role, &password, &email, &created)
	if err != nil {
		log.Println(err)
		return
	}

	key, err := createRecoveryKey(r, tx, id)
	if err != nil {
		log.Println(err)
		return
	}

	err = SendRecovery(r, name, email, key)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

// UpdateUser はユーザ情報を更新する関数
func UpdateUser(r *http.Request, user User) (User, error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err := errors.New("DB instance not found.")
		return user, err
	}

	tx, err := db.Begin()
	if err != nil {
		return user, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	rslt, err := tx.Exec(sqlUpdateUser, user.Name, user.VoiceChatID, user.Role, user.Email, user.DeleteFlag, user.Id)
	if err != nil {
		return user, err
	}
	if cnt, err := rslt.RowsAffected(); err != nil {
		return user, err
	} else {
		log.Println(cnt)
	}
	return user, nil
}

// UpdateUserStatus はユーザステータスを更新する関数
func UpdateUserStatus(r *http.Request, userId int32, status string) error {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err := errors.New("DB instance not found.")
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	now := time.Now()
	rslt, err := tx.Exec(sqlUpdateUserStatus, userId, status, now, status, now)
	if err != nil {
		return err
	}
	if _, err := rslt.RowsAffected(); err != nil {
		return err
	}
	return nil
}

// UpdatePasswordByRecoveryKey はパスワードを変更する関数
func UpdatePasswordByRecoveryKey(r *http.Request, key string, passwd string) (err error) {
	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var (
		t, c time.Time
		uid  int32
	)
	err = tx.QueryRow(sqlFindUserPasswdRecovery, key).Scan(&uid, &t, &c)
	if err != nil {
		log.Println(err)
		return
	}
	if t.Unix() < time.Now().Unix()-1800 {
		err = errors.New("Recovery key was expired.")
		log.Println(err, t.Unix(), time.Now().Unix())
		return
	}

	err = updatePasswordById(r, tx, uid, passwd, c)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = tx.Exec(sqlDeleteUserPasswdRecovery, key)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// UpdatePasswordByAuthId はパスワードを変更する関数
func UpdatePasswordByAuthId(r *http.Request, authId string, curPasswd string, newPasswd string) (err error) {
	u, err := Authenticate(r, authId, curPasswd)
	if err != nil {
		log.Println(err)
		if err == ErrUnauthorized {
			err = ErrNotFound
		}
		return
	}

	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = updatePasswordById(r, tx, u.Id, newPasswd, u.Created)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// updatePasswordById はパスワードを変更する関数
func updatePasswordById(r *http.Request, tx *sql.Tx, id int32, passwd string, created time.Time) (err error) {
	p, err := hashPassword(passwd, created)
	if err != nil {
		log.Println(err)
		return
	}
	rslt, err := tx.Exec(sqlUpdatePasswd, p, id)
	if err != nil {
		log.Println(err)
		return
	}
	var cnt int64
	if cnt, err = rslt.RowsAffected(); err != nil {
		log.Println(err)
		return
	} else if cnt == 0 {
		err = errors.New("Your account is deleted.")
		log.Println(err)
		return
	}
	return
}

// DelUser はユーザを削除する関数
func DelUser(r *http.Request, userId int32) (err error) {

	db, ok := context.Get(r, dbkey).(*sql.DB)
	if !ok {
		err = errors.New("DB instance not found.")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.Exec(sqlDeleteUser, userId)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// uuid は UUID version 4 実装
func uuid() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

type SystemConf struct {
	Name string
	URL  *url.URL
	Mail *mail.Address
}
