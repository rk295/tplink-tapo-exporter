package tapo

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	timeout = time.Second * 2

	tapoTimeFormat     = "2006-01-02 15:04:05"
	defaultContentType = "application/json"

	defaultAPIPath  = "app"
	defaultScheme   = "http"
	defaultTokenKey = "token"

	methodSecurePassThrough = "securePassthrough"
	methodHandshake         = "handshake"
	methodDeviceLogin       = "login_device"
	methodSetDeviceInfo     = "set_device_info"
	methodGetDeviceInfo     = "get_device_info"
	methodGetEnergyUsage    = "get_energy_usage"
	methodGetDeviceUsage    = "get_device_usage"

	// TapoIPEnvName is a convenience constant that can be used as an
	// environment variable name for configuring the client with an IP address.
	// Supports ip:port notation.
	TapoIPEnvName = "TAPO_IP"

	// TapoEmailEnvName is a convenience constant that can be used as an
	// environment variable name for configuring the client with an email address.
	TapoEmailEnvName = "TAPO_EMAIL"

	// TapoPasswordEnvName is a convenience constant that can be used as an
	// environment variable name for configuring the client with a password.
	TapoPasswordEnvName = "TAPO_PASSWORD"

	// tapoSmartPlug is the string returned for "Type" if its a Smart plug with
	// energy monitoring
	tapoSmartPlug = "SMART.TAPOPLUG"
)

var (
	errorNoLogin = errors.New("login was not performed")
)

//
// Public functions
//

// New returns a new Tapo device configured with the provided ip, email, password.
// ip can be in the form of ip:port.
func New(ip, email, password string) *Device {
	h := sha1.New()
	h.Write([]byte(email))
	digest := hex.EncodeToString(h.Sum(nil))
	encodedEmail := base64.StdEncoding.EncodeToString([]byte(digest))
	encodedPassword := base64.StdEncoding.EncodeToString([]byte(password))

	return &Device{
		ip:              ip,
		encodedEmail:    encodedEmail,
		encodedPassword: encodedPassword,
		client:          &http.Client{Timeout: timeout},
	}
}

// NewFromEnv returns a new Tapo device configured from the environment, using
// the Tapo..... constants provided by the package.
func NewFromEnv() (*Device, error) {
	ip := os.Getenv(TapoIPEnvName)
	email := os.Getenv(TapoEmailEnvName)
	password := os.Getenv(TapoPasswordEnvName)

	if ip == "" || email == "" || password == "" {
		return &Device{}, fmt.Errorf("must set %s, %s, %s environment variables", TapoIPEnvName, TapoEmailEnvName, TapoPasswordEnvName)
	}

	return New(ip, email, password), nil
}

// Login performs the actual login to the device.
func (d *Device) Login() (err error) {
	if d.cipher == nil {
		err := d.handshake()
		if err != nil {
			return err
		}
	}

	req := &jsonReq{
		Method: methodDeviceLogin,
		Params: loginRequest{
			Username: d.encodedEmail,
			Password: d.encodedPassword,
		},
	}

	loginResponse := &loginResponse{}

	if err := d.req(req, &loginResponse); err != nil {
		return err
	}

	d.token = &loginResponse.Token
	return nil
}

func (d *Device) SetDeviceInfo(params map[string]interface{}) (err error) {

	req := &jsonReq{
		Method: methodSetDeviceInfo,
		Params: params,
	}
	jsonResp := &jsonResp{}

	err = d.req(req, &jsonResp)
	return err
}

func (d *Device) Switch(status bool) (err error) {
	return d.SetDeviceInfo(map[string]interface{}{
		"device_on": status,
	})
}

func (d *Device) GetDeviceInfo() (*Status, error) {
	status := &Status{}
	if err := d.req(&jsonReq{Method: methodGetDeviceInfo}, &status); err != nil {
		return status, err
	}

	// Base64 decode the Nickname and SSID of the device to be helpful to users
	// of this module
	nicknameEncoded, err := base64.StdEncoding.DecodeString(status.Nickname)
	if err != nil {
		return status, err
	}
	status.Nickname = string(nicknameEncoded)

	SSIDEncoded, err := base64.StdEncoding.DecodeString(status.SSID)
	if err != nil {
		return status, err
	}
	status.SSID = string(SSIDEncoded)

	return status, nil
}

func (d *Device) GetEnergyUsage() (*EnergyInfo, error) {
	energyInfo := EnergyInfo{}
	if err := d.req(&jsonReq{Method: methodGetEnergyUsage}, &energyInfo); err != nil {
		return &energyInfo, err
	}
	return &energyInfo, nil
}

func (d *Device) GetDeviceUsage() (*DeviceUsage, error) {
	deviceUsage := DeviceUsage{}
	if err := d.req(&jsonReq{Method: methodGetDeviceUsage}, &deviceUsage); err != nil {
		return &deviceUsage, err
	}
	return &deviceUsage, nil
}

// EmeterSupported returns true if the plug supports energy monitoring
func (s *Status) EmeterSupported() bool {
	return s.Type == tapoSmartPlug
}

//
// Private functions
//
func (d *Device) req(p *jsonReq, target interface{}) error {
	if d.token == nil && p.Method != methodDeviceLogin {
		return errorNoLogin
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}

	apiResponse := &apiResponse{
		Result: &target,
	}

	reply, err := d.doRequest(payload)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewBuffer(reply)).Decode(apiResponse); err != nil {
		return err
	}

	if err = d.checkErrorCode(apiResponse.ErrorCode); err != nil {
		return err
	}

	return nil
}

func (d *Device) getURL() string {
	u := &url.URL{
		Scheme: defaultScheme,
		Host:   d.ip,
		Path:   defaultAPIPath,
	}

	if d.token != nil {
		q := u.Query()
		q.Set(defaultTokenKey, *d.token)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func (d *Device) doRequest(payload []byte) ([]byte, error) {
	encryptedPayload := base64.StdEncoding.EncodeToString(d.cipher.encrypt(payload))

	securedPayloadReq := &jsonReq{
		Method: methodSecurePassThrough,
		Params: securePassThroughRequest{
			Request: encryptedPayload,
		},
	}

	securedPayload, err := json.Marshal(securedPayloadReq)
	if err != nil {
		return []byte{}, err
	}

	req, err := http.NewRequest("POST", d.getURL(), bytes.NewBuffer(securedPayload))
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("Cookie", d.sessionID)
	req.Close = true

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	jsonResp := &jsonResp{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return nil, err
	}

	switch jsonResp.ErrorCode {
	case 9999:
		if err = d.handshake(); err != nil {
			return nil, err
		}
		if err = d.Login(); err != nil {
			return nil, err
		}

		return d.doRequest(payload)
	default:
		if err = d.checkErrorCode(jsonResp.ErrorCode); err != nil {
			return nil, err
		}
	}

	encryptedResponse, err := base64.StdEncoding.DecodeString(jsonResp.Result.Response)
	if err != nil {
		return nil, err
	}

	return d.cipher.decrypt(encryptedResponse), nil
}

func (d *Device) checkErrorCode(errorCode int) error {
	if errorCode != 0 {
		return fmt.Errorf("error code %d", errorCode)
	}

	return nil
}

func (d *Device) handshake() (err error) {
	privKey, pubKey := generateRSAKeys()

	pubPEM := dumpRSAPEM(pubKey)

	req := &jsonReq{
		Method: methodHandshake,
		Params: handshakeRequest{
			Key:             string(pubPEM),
			RequestTimeMils: 0,
		},
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return
	}

	resp, err := http.Post(d.getURL(), defaultContentType, bytes.NewBuffer(payload))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	jsonResp := &jsonResp{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return err
	}

	if err = d.checkErrorCode(jsonResp.ErrorCode); err != nil {
		return
	}

	encryptedEncryptionKey, err := base64.StdEncoding.DecodeString(jsonResp.Result.Key)
	if err != nil {
		return err
	}

	encryptionKey, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, encryptedEncryptionKey)
	if err != nil {
		return err
	}
	d.cipher = &Cipher{
		key: encryptionKey[:16],
		iv:  encryptionKey[16:],
	}

	d.sessionID = strings.Split(resp.Header.Get("Set-Cookie"), ";")[0]

	return
}
