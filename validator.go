package mizumanju

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
)

// ErrValidation は入力チェックエラーであることを表す
var ErrValidation error = errors.New("Bad Request")

// validateLogin は logionParams の入力チェックをする関数
func validateLogin(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	lp, ok := p.(*loginParams)
	if !ok {
		err = fmt.Errorf("Expected *loginParams, but actual is %T", p)
		log.Println(err)
		return
	}

	m := make(map[string][]string)

	if lp.Username == "" {
		m["username"] = []string{"Username is required."}
	}
	if lp.Password == "" {
		m["password"] = []string{"Password is required."}
	}

	if len(m) > 0 {
		b, err = json.Marshal(NewResponse(m, nil))
		if err != nil {
			log.Println(err)
			return
		}
		return b, ErrValidation
	}
	return
}

// validateUsersPassword は recoveryParams の入力チェックをする関数
func validateRecovery(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	pp, ok := p.(*recoveryParams)
	if !ok {
		err = fmt.Errorf("Expected *recoveryParams, but actual is %T", p)
		log.Println(err)
		return
	}

	m := make(map[string][]string)

	if pp.Password == "" {
		m["password"] = []string{"Password is required."}
	}

	if len(m) > 0 {
		b, err = json.Marshal(NewResponse(m, nil))
		if err != nil {
			log.Println(err)
			return
		}
		return b, ErrValidation
	}
	return
}

// validateUsersPost は User の入力チェックをする関数
func validateUser(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	u, ok := p.(*User)
	if !ok {
		err = fmt.Errorf("Expected *User, but actual is %T", p)
		log.Println(err)
		return
	}

	m := make(map[string][]string)

	if u.Id == 0 && u.AuthId == "" {
		m["authId"] = []string{"Auth ID is required."}
	}
	if u.Name == "" {
		m["name"] = []string{"Name is required."}
	}
	if u.Role == "" {
		m["role"] = []string{"Role is required."}
	} else if u.Role != admin && u.Role != editor {
		m["role"] = []string{"Role is invalid."}
	}
	if u.Email == "" {
		m["email"] = []string{"Email is required."}
	} else {
		addr, err := mail.ParseAddress(u.Email)
		if err != nil {
			m["email"] = []string{err.Error()}
		} else {
			u.Email = addr.Address
		}
	}

	if len(m) > 0 {
		b, err = json.Marshal(NewResponse(m, nil))
		if err != nil {
			log.Println(err)
			return
		}
		return b, ErrValidation
	}
	return
}

// validateRecoveryRequest は recoveryRequestParams の入力チェックをする関数
func validateRecoveryRequest(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	rp, ok := p.(*recoveryRequestParams)
	if !ok {
		err = fmt.Errorf("Expected *recoveryRequestParams, but actual is %T", p)
		log.Println(err)
		return
	}

	m := make(map[string][]string)

	if rp.Email == "" {
		m["email"] = []string{"Email is required."}
	} else {
		addr, err := mail.ParseAddress(rp.Email)
		if err != nil {
			m["email"] = []string{err.Error()}
		} else {
			rp.Email = addr.Address
		}
	}

	if len(m) > 0 {
		b, err = json.Marshal(NewResponse(m, nil))
		if err != nil {
			log.Println(err)
			return
		}
		return b, ErrValidation
	}
	return
}

// validatePassword は passwordParams の入力チェックをする関数
func validatePassword(w http.ResponseWriter, r *http.Request, p params) (b []byte, err error) {
	pp, ok := p.(*passwordParams)
	if !ok {
		err = fmt.Errorf("Expected *passwordParams, but actual is %T", p)
		log.Println(err)
		return
	}

	m := make(map[string][]string)
	if pp.NewPassword == "" {
		m["newPassword"] = []string{"New password is required."}
	}
	if pp.CurrentPassword == "" {
		m["currentPassword"] = []string{"Current password is required."}
	}

	if len(m) > 0 {
		b, err = json.Marshal(NewResponse(m, nil))
		if err != nil {
			log.Println(err)
			return
		}
		return b, ErrValidation
	}
	return
}
