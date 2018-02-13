# Craas

Card reading as a service. Craas presents a gRPC service, `CardReader`, that publishes card read events to gRPC subscribers.

- [X] client debugging. `craas -testing` sets up a REPL so that you can publish arbitrary card events from the command line. Useful for testing your subscriber implementation
- [ ] read card events from a serial device

## Usage

[Download](https://github.com/IQ-Inc/craas/releases) the relevant binary for your system, or build from source.

```bash
$ ./crass -h # print help
$ ./craas -serial /dev/ttyUSB0 -port :8080 # read from /dev/ttyUSB0, and publish to localhost port 8080
$ ./craas -testing -port 127.0.0.1:9999 # read from the testing REPL, and publish to 127.0.0.1:9999
```