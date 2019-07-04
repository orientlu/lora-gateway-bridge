package config

import (
	"time"
)

// Config defines the configuration structure.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	}

	Backend struct {
		Type string `mapstructure:"type"`

		SemtechUDP struct {
			UDPBind       string `mapstructure:"udp_bind"`
			SkipCRCCheck  bool   `mapstructure:"skip_crc_check"`
			FakeRxTime    bool   `mapstructure:"fake_rx_time"`
			Configuration []struct {
				GatewayID      string `mapstructure:"gateway_id"`
				BaseFile       string `mapstructure:"base_file"`
				OutputFile     string `mapstructure:"output_file"`
				RestartCommand string `mapstructure:"restart_command"`
			} `mapstructure:"configuration"`
		} `mapstructure:"semtech_udp"`

		BasicStation struct {
			Bind         string        `mapstructure:"bind"`
			TLSCert      string        `mapstructure:"tls_cert"`
			TLSKey       string        `mapstructure:"tls_key"`
			CACert       string        `mapstructure:"ca_cert"`
			PingInterval time.Duration `mapstructure:"ping_interval"`
			ReadTimeout  time.Duration `mapstructure:"read_timeout"`
			WriteTimeout time.Duration `mapstructure:"write_timeout"`
			Filters      struct {
				NetIDs   []string    `mapstructure:"net_ids"`
				JoinEUIs [][2]string `mapstructure:"join_euis"`
			} `mapstructure:"filters"`
			Region       string `mapstructure:"region"`
			FrequencyMin uint32 `mapstructure:"frequency_min"`
			FrequencyMax uint32 `mapstructure:"frequency_max"`
		} `mapstructure:"basic_station"`
	} `mapstructure:"backend"`

	Integration struct {
		Marshaler string `mapstructure:"marshaler"`

		MQTT struct {
			EventTopicTemplate   string `mapstructure:"event_topic_template"`
			CommandTopicTemplate string `mapstructure:"command_topic_template"`
			NotifyTopicTemplate  string `mapstructure:"notify_topic_template"`

			Auth struct {
				Type string `mapstructure:"type"`

				Generic struct {
					Server               string        `mapstructure:"server"`
					Username             string        `mapstructure:"username"`
					Password             string        `mapstrucure:"password"`
					CACert               string        `mapstructure:"ca_cert"`
					TLSCert              string        `mapstructure:"tls_cert"`
					TLSKey               string        `mapstructure:"tls_key"`
					QOS                  uint8         `mapstructure:"qos"`
					CleanSession         bool          `mapstructure:"clean_session"`
					ClientID             string        `mapstructure:"client_id"`
					MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval"`
				} `mapstructure:"generic"`

				GCPCloudIoTCore struct {
					Server        string        `mapstructure:"server"`
					DeviceID      string        `mapstructure:"device_id"`
					ProjectID     string        `mapstructure:"project_id"`
					CloudRegion   string        `mapstructure:"cloud_region"`
					RegistryID    string        `mapstructure:"registry_id"`
					JWTExpiration time.Duration `mapstructure:"jwt_expiration"`
					JWTKeyFile    string        `mapstructure:"jwt_key_file"`
				} `mapstructure:"gcp_cloud_iot_core"`

				AzureIoTHub struct {
					DeviceConnectionString string        `mapstructure:"device_connection_string"`
					DeviceID               string        `mapstructure:"-"`
					Hostname               string        `mapstructure:"-"`
					DeviceKey              string        `mapstructure:"-"`
					SASTokenExpiration     time.Duration `mapstructure:"sas_token_expiration"`
				} `mapstructure:"azure_iot_hub"`
			} `mapstructure:"auth"`
		} `mapstructure:"mqtt"`
	} `mapstructure:"integration"`

	Metrics struct {
		Prometheus struct {
			EndpointEnabled bool   `mapstructure:"endpoint_enabled"`
			Bind            string `mapstructure:"bind"`
		}
	}

	MetaData struct {
		Static  map[string]string `mapstructure:"static"`
		Dynamic struct {
			ExecutionInterval    time.Duration     `mapstructure:"execution_interval"`
			MaxExecutionDuration time.Duration     `mapstructure:"max_execution_duration"`
			Commands             map[string]string `mapstructure:"commands"`
		} `mapstructure:"dynamic"`
	} `mapstructure:"meta_data"`
}

// C holds the global configuration.
var C Config
