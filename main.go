package main

import (
	"fmt"
	"github.com/go-ping/ping"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var opts struct {
	Interval      int    `long:"interval" description:"Interval in milliseconds" default:"3000"`
	MaxFailures   int    `long:"max-failures" description:"Maximum number of consecutive failures before running the command" default:"5"`
	IP            string `long:"host" description:"Host name to ping (IP address recommended)" default:"1.1.1.1"`
	Timeout       int    `long:"timeout" description:"Timeout in milliseconds" default:"1000"`
	Unpriviledged bool   `long:"unprivileged" description:"Run ping in unprivileged mode (see https://github.com/go-ping/ping) "`
	Cooldown      int    `long:"cooldown" description:"Cooldown in milliseconds after running the command before pinging again" default:"300000"`
	Command       struct {
		Args []string `required:"1"`
	} `positional-args:"yes"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if terminalOutput() {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().Timestamp().
			Logger()
	}
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(1)
		}
		log.Logger.Fatal().Err(err).Msg("Failed to parse command line arguments")
	}

	pinger, err := ping.NewPinger(opts.IP)
	if err != nil {
		log.Logger.Fatal().Err(err).Msgf("Failed to create pinger for %s", opts.IP)
	}
	pinger.Count = 1
	pinger.Timeout = time.Duration(opts.Timeout) * time.Millisecond
	pinger.SetPrivileged(!opts.Unpriviledged)

	failureCount := 0
	for {
		err = pingIP(pinger)
		if err != nil {
			failureCount++
			log.Warn().Err(err).Msgf("Failed to ping %s. Failure count: %d", opts.IP, failureCount)
		} else {
			log.Info().Msgf("Successfully pinged %s", opts.IP)
			failureCount = 0
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

func pingIP(pinger *ping.Pinger) error {

	err := pinger.Run()
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to ping %s", pinger.Addr())
	}
	stats := pinger.Statistics()

	if stats.PacketsRecv == 0 {
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
