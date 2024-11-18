package main

import (
	"fmt"
	"github.com/SourceForgery/duc2mqtt/hassio"
	"github.com/go-ping/ping"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/url"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"
)

var opts struct {
	Verbose       []bool `long:"verbose" short:"v" description:"Verbosity. Repeat for increasing"`
	Quiet         []bool `long:"quiet" short:"q" description:"Make less verbose. Repeat for even less"`
	Version       bool   `long:"version" short:"V" description:"Display version"`
	LoggingFormat string `short:"l" long:"logging" choice:"coloured" choice:"plain" choice:"json" choice:"default" default:"default" description:"Log output format"`

	Interval        int    `long:"interval" env:"INTERVAL" description:"Interval in milliseconds" default:"3000"`
	MaxFailures     int    `long:"max-failures" env:"MAX_FAILURES" description:"Maximum number of consecutive failures before running the command" default:"5"`
	IP              string `long:"host" env:"IP" description:"Host name to ping (IP address recommended)" default:"1.1.1.1"`
	Timeout         int    `long:"timeout" env:"TIMEOUT" description:"Timeout in milliseconds" default:"1000"`
	Unprivileged    bool   `long:"unprivileged" env:"UNPRIVILEGED" description:"Run ping in unprivileged mode (see https://github.com/go-ping/ping) "`
	Cooldown        int    `long:"cooldown" env:"COOLDOWN" description:"Cooldown in milliseconds after running the command before pinging again" default:"300000"`
	MqttUri         string `long:"mqtt-uri" env:"MQTT_URI" description:"The URI to the mqtt server. If using virtual-host extension (rabbitmq), it would be tcp://test.mosquitto.org:1883/vhost"`
	MqttUniqueId    string `long:"mqtt-unique-id" env:"MQTT_UNIQUE_ID" description:"The unique id of this device" default:"network-monitor"`
	MqttName        string `long:"mqtt-name" env:"MQTT_NAME" description:"The name of this device" default:"network-monitor"`
	MqttTopicPrefix string `long:"mqtt-topic-prefix" env:"MQTT_TOPIC_PREFIX" description:"The mqtt prefix to use" default:"homeassistant"`
	Command         struct {
		Args []string `required:"1"`
	} `positional-args:"yes"`
}

var version = "unknown"
var hassioClient *hassio.Client
var problem *bool
var hassioSensor = hassio.NewAlarmSensorConfig("internet", "uptime")

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(1)
		}
		log.Logger.Fatal().Err(err).Msg("Failed to parse command line arguments")
	}

	initializeLogging()

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatal().Msg("Failed to read build info")
	}

	var vcsVersion = "unknown"
	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			vcsVersion = setting.Value
		}
	}

	if opts.Version {
		log.Info().
			Str("vcsVersion", vcsVersion).
			Str("goVersion", buildInfo.GoVersion).
			Str("version", version).
			Msgf("duc2mqtt version %s compiled with %s, commitId: %s", version, buildInfo.GoVersion, vcsVersion)
		os.Exit(1)
	}

	pinger, err := ping.NewPinger(opts.IP)
	if err != nil {
		log.Logger.Fatal().Err(err).Msgf("Failed to create pinger for %s", opts.IP)
	}
	pinger.Count = 1
	pinger.Timeout = time.Duration(opts.Timeout) * time.Millisecond
	pinger.SetPrivileged(!opts.Unprivileged)

	if opts.MqttUri != "" {
		setupMqtt()
	}

	failureCount := 0
	for {
		err = pingIP(pinger)
		if err != nil {
			failureCount++
			log.Warn().Err(err).Msgf("Failed to ping %s. Failure count: %d", opts.IP, failureCount)
			setStatus(false)
		} else {
			log.Debug().Msgf("Successfully pinged %s", opts.IP)
			failureCount = 0
			setStatus(true)
		}

		if failureCount >= opts.MaxFailures {
			log.Error().Msgf("Ping to %s failed %d times consecutively. Running command: %s", opts.IP, opts.MaxFailures, strings.Join(opts.Command.Args, " "))
			runCommand(opts.Command.Args)
			failureCount = 0
			time.Sleep(time.Duration(opts.Cooldown) * time.Millisecond)
		}

		time.Sleep(time.Duration(opts.Interval) * time.Millisecond)
	}
}

func setupMqtt() {
	mqttUrl, err := url.Parse(opts.MqttUri)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse mqtt url")
	}

	amqpVhost := strings.TrimPrefix(mqttUrl.Path, "/")
	hassioClient, err = hassio.ConnectMqtt(*mqttUrl, amqpVhost, opts.MqttUniqueId, opts.MqttTopicPrefix)

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatal().Msg("Failed to read build info")
	}

	hassioClient.Device = &hassio.Device{
		Identifiers:      []string{opts.MqttUniqueId},
		Name:             opts.MqttName,
		SWVersion:        buildInfo.Main.Version,
		HWVersion:        "N/A",
		SerialNumber:     "N/A",
		Model:            "NetworkMonitor",
		ModelID:          "NetworkMonitor",
		Manufacturer:     "SourceForgery",
		ConfigurationURL: fmt.Sprintf("http://localhost/"),
	}
	hassioClient.SensorConfigurationData = map[string]hassio.SensorConfig{
		"internet": hassioSensor,
	}

	err = hassioClient.SubscribeToHomeAssistantStatus()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to subscribe to Home Assistant status")
	}
}

func pingIP(pinger *ping.Pinger) error {
	oldStats := pinger.Statistics()

	err := pinger.Run()
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to ping %s", pinger.Addr())
	}
	stats := pinger.Statistics()
	pinger.Stop()

	if oldStats.PacketsRecv < stats.PacketsRecv {
		return fmt.Errorf("no packets received")
	}
	return nil
}

func runCommand(cmd []string) {
	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to run command: %s", cmd)
	}
	log.Info().Msgf("Command output: %s", out)
}

func terminalOutput() bool {
	o, _ := os.Stdout.Stat()
	return (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func setStatus(newStatus bool) {
	if problem != nil && newStatus == *problem {
		return
	}
	problem = &newStatus
	if hassioClient != nil {
		value := 1.0
		if newStatus {
			value = 0
		}
		err := hassioClient.SendSensorData(hassioSensor.SensorType(), map[string]string{
			hassioSensor.SensorId(): hassioSensor.ConvertValue(value),
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to send data")
		}
	}
}

func initializeLogging() {
	log.Logger = getLogger(opts.LoggingFormat).
		With().Timestamp().
		Logger()
	setLogLevel(len(opts.Quiet) - len(opts.Verbose) + 1)
}

func getLogger(loggingFormat string) (lg zerolog.Logger) {
	switch loggingFormat {
	case "default":
		if terminalOutput() {
			return getLogger("coloured")
		} else {
			return getLogger("plain")
		}
	case "json":
		return zerolog.New(os.Stdout)
	case "coloured":
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	case "plain":
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: true})
	default:
		log.Panic().Msgf("What the f is %s", loggingFormat)
	}
	return lg
}

func setLogLevel(verbosity int) {
	if verbosity < 0 {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if verbosity > 6 {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else {
		zerolog.SetGlobalLevel(zerolog.Level(verbosity))
	}
}
