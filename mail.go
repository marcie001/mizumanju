package mizumanju

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"text/template"

	"github.com/gorilla/context"
)

const (
	// 招待メールテンプレート
	defaultInvitationMail = `Subject: Welcome to {{.SystemName}}!!

Hi, {{.ToName}}. You are invited to {{.SystemName}} from {{.FromName}}.
Please visit below url and set your password within 30 minutes.

{{.RecoveryURL}}

--
{{.SystemName}}
{{.SystemURL}}`
	// パスワードリカバリメールテンプレート
	defaultRecoveryMail = `Subject: Password Recovery

Please visit below url and set your password within 30 minutes.

{{.RecoveryURL}}

--
{{.SystemName}}
{{.SystemURL}}`
)

var (
	recoveryPathFormat = "/recovery/%s"
)

// SetMailTmpl は Template インスタンスを context に保存する関数。
func SetMailTmpl(r *http.Request, tmpl *template.Template) {
	context.Set(r, tmplkey, tmpl)
}

// SetSmtpConf は SmtpConf インスタンスを context に保存する関数。
func SetSmtpConf(r *http.Request, smtpConf *SmtpConf) {
	context.Set(r, smtpkey, smtpConf)
}

// CreateTemplate は Template インスタンスを作成する関数。
func CreateTemplate() *template.Template {
	tmpl := template.Must(template.New("invitationMail").Parse(defaultInvitationMail))
	tmpl = template.Must(tmpl.New("recoveryMail").Parse(defaultRecoveryMail))
	return tmpl
}

// SendInvitation は招待メールを送信する関数
func SendInvitation(r *http.Request, toName string, toAddress string, recoveryKey string) error {
	tmpl, ok := context.Get(r, tmplkey).(*template.Template)
	if !ok {
		return errors.New("Template instance not found.")
	}
	scnf, ok := context.Get(r, systemkey).(*SystemConf)
	if !ok {
		return errors.New("SystemConf instance not found.")
	}
	to := &mail.Address{toName, toAddress}
	url := scnf.URL.ResolveReference(&url.URL{Fragment: fmt.Sprintf(recoveryPathFormat, recoveryKey)})
	d := InvitationData{
		SystemName:  scnf.Name,
		SystemURL:   scnf.URL.String(),
		RecoveryURL: url.String(),
		ToName:      toName,
		FromName:    scnf.Name,
	}

	return send(r, scnf.Mail.Address, to.Address, func(r *http.Request, wc io.WriteCloser) error {
		return tmpl.ExecuteTemplate(wc, "invitationMail", d)
	})
}

// SendRecovery は招待メールを送信する関数
func SendRecovery(r *http.Request, toName string, toAddress string, recoveryKey string) error {
	tmpl, ok := context.Get(r, tmplkey).(*template.Template)
	if !ok {
		return errors.New("Template instance not found.")
	}
	scnf, ok := context.Get(r, systemkey).(*SystemConf)
	if !ok {
		return errors.New("SystemConf instance not found.")
	}
	to := &mail.Address{toName, toAddress}
	url := scnf.URL.ResolveReference(&url.URL{Fragment: fmt.Sprintf(recoveryPathFormat, recoveryKey)})
	d := RecoveryData{
		SystemName:  scnf.Name,
		SystemURL:   scnf.URL.String(),
		RecoveryURL: url.String(),
	}

	return send(r, scnf.Mail.Address, to.Address, func(r *http.Request, wc io.WriteCloser) error {
		return tmpl.ExecuteTemplate(wc, "recoveryMail", d)
	})
}

type writeBody func(r *http.Request, wc io.WriteCloser) error

func send(r *http.Request, from, to string, wb writeBody) (err error) {
	scnf, ok := context.Get(r, smtpkey).(*SmtpConf)
	if !ok {
		return errors.New("SmtpConf instance not found.")
	}
	c, err := smtp.Dial(scnf.Addr())
	if err != nil {
		return
	}
	defer func() {
		if cerr := c.Quit(); err == nil {
			err = cerr
		}
	}()

	if scnf.TLS {
		tcnf := &tls.Config{ServerName: scnf.Host}
		if err = c.StartTLS(tcnf); err != nil {
			return
		}
	}

	if scnf.User != "" {
		auth := smtp.PlainAuth("", scnf.User, scnf.Password, scnf.Host)
		err = c.Auth(auth)
		if err != nil {
			return err
		}
	}

	if err = c.Mail(from); err != nil {
		log.Println(from)
		return
	}
	if err = c.Rcpt(to); err != nil {
		return
	}
	wc, err := c.Data()
	if err != nil {
		return
	}
	defer func() {
		if cerr := wc.Close(); err == nil {
			err = cerr
		}
	}()
	if err = wb(r, wc); err != nil {
		return
	}
	return
}

type RecoveryData struct {
	SystemName, SystemURL, RecoveryURL string
}

type InvitationData struct {
	SystemName, SystemURL, FromName, ToName, RecoveryURL string
}

type SmtpConf struct {
	Host, Sender, User, Password string
	Port                         int
	TLS                          bool
}

func (scnf *SmtpConf) Addr() string {
	return fmt.Sprintf("%s:%d", scnf.Host, scnf.Port)
}
