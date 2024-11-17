[![ci](https://github.com/SourceForgery/network-monitor/actions/workflows/ci.yml/badge.svg)](https://github.com/SourceForgery/network-monitor/actions/workflows/ci.yml)
# Network Monitor Tool

This is a Network Monitor Tool that pings a specified host at regular intervals and executes a predefined command if the host becomes unreachable for a given number of consecutive failures.

## Features

- **Customizable Ping Interval**: Specify the interval between pings in milliseconds.
- **Max Failure Threshold**: Set the maximum number of consecutive ping failures before triggering the command.
- **Ping Timeout**: Configure the time to wait for a ping response.
- **Unprivileged Mode**: Option to run ping in unprivileged mode.
- **Cooldown Period**: Define a cooldown period after the command is executed before starting to ping again.
- **Logging**: Provides detailed logging with timestamps, errors, and information messages.

## Installation

1. **Clone the repository**:
    ```sh
    git clone <repository-url>
    cd <repository-directory>
    ```

2. **Build the application**:
    ```sh
    go build -o network-monitor
    ```

## Usage

Run the executable with the required parameters:

```sh
./network-monitor [OPTIONS] COMMAND...
```


### Options

- `--interval`: Interval in milliseconds between pings (default: 3000 ms).
- `--max-failures`: Maximum number of consecutive ping failures before running the command (default: 5).
- `--host`: Hostname to ping (IP address recommended, default: `1.1.1.1`).
- `--timeout`: Timeout in milliseconds for each ping (default: 1000 ms).
- `--unprivileged`: Run ping in unprivileged mode.
- `--cooldown`: Cooldown period in milliseconds after running the command before pinging again (default: 300000 ms).

### Positional Arguments

- `COMMAND`: The command to execute if the max failures threshold is reached. Pass the command and its arguments as positional arguments.

### Example

To ping `8.8.8.8` every 3 seconds and run `echo 'Host unreachable'` if it fails to ping successfully 5 consecutive times:

```sh
./network-monitor --host 8.8.8.8 --interval 3000 --max-failures 5 --timeout 1000 --cooldown 60000 echo Host unreachable
```

## Logging

Logs are printed to the console with timestamps and include information on:

- Successful pings
- Ping failures
- Command execution status and output

## Dependencies

- `github.com/go-ping/ping`
- `github.com/jessevdk/go-flags`
- `github.com/rs/zerolog`

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

I think I'm done with this program, and I'm not planning on extending it, so please do fork, but don't expect
there to be any activity in this repo beyond this initial push.

## Contact

For questions or issues, please open an issue on the repository's issue tracker.

---

Above is almost exclusively auto-generated content, so there may be errors. Happy monitoring! 📡

