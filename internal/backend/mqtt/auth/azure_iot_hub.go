package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// See:
// https://docs.microsoft.com/en-us/azure/iot-hub/iot-hub-mqtt-support#tlsssl-configuration
// https://github.com/Azure/azure-iot-sdk-c/blob/master/certs/certs.c
const digiCertBaltimoreRootCA = `
-----BEGIN CERTIFICATE-----
MIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ
RTESMBAGA1UEChMJQmFsdGltb3JlMRMwEQYDVQQLEwpDeWJlclRydXN0MSIwIAYD
VQQDExlCYWx0aW1vcmUgQ3liZXJUcnVzdCBSb290MB4XDTAwMDUxMjE4NDYwMFoX
DTI1MDUxMjIzNTkwMFowWjELMAkGA1UEBhMCSUUxEjAQBgNVBAoTCUJhbHRpbW9y
ZTETMBEGA1UECxMKQ3liZXJUcnVzdDEiMCAGA1UEAxMZQmFsdGltb3JlIEN5YmVy
VHJ1c3QgUm9vdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKMEuyKr
mD1X6CZymrV51Cni4eiVgLGw41uOKymaZN+hXe2wCQVt2yguzmKiYv60iNoS6zjr
IZ3AQSsBUnuId9Mcj8e6uYi1agnnc+gRQKfRzMpijS3ljwumUNKoUMMo6vWrJYeK
mpYcqWe4PwzV9/lSEy/CG9VwcPCPwBLKBsua4dnKM3p31vjsufFoREJIE9LAwqSu
XmD+tqYF/LTdB1kC1FkYmGP1pWPgkAx9XbIGevOF6uvUA65ehD5f/xXtabz5OTZy
dc93Uk3zyZAsuT3lySNTPx8kmCFcB5kpvcY67Oduhjprl3RjM71oGDHweI12v/ye
jl0qhqdNkNwnGjkCAwEAAaNFMEMwHQYDVR0OBBYEFOWdWTCCR1jMrPoIVDaGezq1
BE3wMBIGA1UdEwEB/wQIMAYBAf8CAQMwDgYDVR0PAQH/BAQDAgEGMA0GCSqGSIb3
DQEBBQUAA4IBAQCFDF2O5G9RaEIFoN27TyclhAO992T9Ldcw46QQF+vaKSm2eT92
9hkTI7gQCvlYpNRhcL0EYWoSihfVCr3FvDB81ukMJY2GQE/szKN+OMY3EU/t3Wgx
jkzSswF07r51XgdIGn9w/xZchMB5hbgF/X++ZRGjD8ACtPhSNzkE1akxehi/oCr0
Epn3o0WC4zxe9Z2etciefC7IpJ5OCBRLbf1wbWsaY71k5h+3zvDyny67G7fyUIhz
ksLi4xaNmjICq44Y3ekQEe5+NauQrz4wlHrQMz2nZQ/1/I6eYs9HRCwBXbsdtTLS
R9I4LtD+gdwyah617jzV/OeBHRnDJELqYzmp
-----END CERTIFICATE-----
`

// AzureIoTHubConfig defines the Azure IoT Hub configuration.
type AzureIoTHubConfig struct {
	DeviceConnectionString string        `mapstructure:"device_connection_string"`
	DeviceID               string        `mapstructure:"-"`
	Hostname               string        `mapstructure:"-"`
	DeviceKey              string        `mapstructure:"-"`
	SASTokenExpiration     time.Duration `mapstructure:"sas_token_expiration"`
}

// AzureIoTHubAuthentication implements the Azure IoT Hub authentication.
type AzureIoTHubAuthentication struct {
	clientID  string
	username  string
	deviceKey []byte
	tlsConfig *tls.Config
	config    AzureIoTHubConfig
}

// NewAzureIoTHubAuthentication creates an AzureIoTHubAuthentication.
func NewAzureIoTHubAuthentication(config AzureIoTHubConfig) (Authentication, error) {
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM([]byte(digiCertBaltimoreRootCA)) {
		return nil, errors.New("append ca cert from pem error")
	}

	if config.DeviceConnectionString != "" {
		kvMap, err := parseConnectionString(config.DeviceConnectionString)
		if err != nil {
			return nil, errors.Wrap(err, "parse connection string error")
		}

		for k, v := range kvMap {
			switch k {
			case "HostName":
				config.Hostname = v
			case "DeviceId":
				config.DeviceID = v
			case "SharedAccessKey":
				config.DeviceKey = v
			}
		}
	}

	username := fmt.Sprintf("%s/%s",
		config.Hostname,
		config.DeviceID,
	)

	deviceKeyB, err := base64.StdEncoding.DecodeString(config.DeviceKey)
	if err != nil {
		return nil, errors.Wrap(err, "decode device key error")
	}

	return &AzureIoTHubAuthentication{
		clientID:  config.DeviceID,
		username:  username,
		deviceKey: deviceKeyB,
		tlsConfig: &tls.Config{
			RootCAs: certpool,
		},
		config: config,
	}, nil
}

// Init applies the initial configuration.
func (a *AzureIoTHubAuthentication) Init(opts *mqtt.ClientOptions) error {
	broker := fmt.Sprintf("ssl://%s:8883", a.config.Hostname)
	opts.AddBroker(broker)
	opts.SetClientID(a.clientID)
	opts.SetUsername(a.username)

	return nil
}

// Update updates the authentication options.
func (a *AzureIoTHubAuthentication) Update(opts *mqtt.ClientOptions) error {
	resourceURI := fmt.Sprintf("%s/devices/%s",
		a.config.Hostname,
		a.config.DeviceID,
	)
	token, err := createSASToken(resourceURI, a.deviceKey, a.config.SASTokenExpiration)
	if err != nil {
		return errors.Wrap(err, "create SAS token error")
	}

	opts.SetPassword(token)

	return nil
}

// ReconnectAfter returns a time.Duration after which the MQTT client must re-connect.
// Note: return 0 to disable the periodical re-connect feature.
func (a *AzureIoTHubAuthentication) ReconnectAfter() time.Duration {
	return a.config.SASTokenExpiration
}

func createSASToken(uri string, deviceKey []byte, expiration time.Duration) (string, error) {
	encoded := url.QueryEscape(uri)
	exp := time.Now().Add(expiration).Unix()

	signature := fmt.Sprintf("%s\n%d", encoded, exp)

	mac := hmac.New(sha256.New, deviceKey)
	mac.Write([]byte(signature))
	hash := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	// IoT Hub SAS Token only needs `sr`, `sig` and `se` unlike other Azure services
	token := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%d",
		encoded,
		hash,
		exp,
	)

	return token, nil
}

func parseConnectionString(str string) (map[string]string, error) {
	out := make(map[string]string)
	pairs := strings.Split(str, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("expected two items in: %+v", kv)
		}

		out[kv[0]] = kv[1]
	}

	return out, nil
}
