# TURN Amplification Factor Measurement Tool

This tool measures the amplification factor of a TURN server by sending allocation requests and analyzing response sizes. 

The tool sends unauthenticated requests to trigger 401 (`Unauthenticated`) TURN responses with small delays between requests to avoid overwhelming the server, measures the raw message sizes, including the STUN header and all STUN attributes, and outputs a summary.

## Usage

Basic usage with default settings (localhost:3478, 100 requests).
```bash
go run main.go
```

Specify custom server:
```bash
go run main.go -server turn.example.com:3478
```

Specify number of requests:
```bash
go run main.go -server turn.example.com:3478 -count 50
```

## Example Output

```
TURN Amplification Factor Measurement Tool
==========================================
Target server: 127.0.0.1:3478
Request count: 100

Results Summary
===============
Successful requests: 100

Overall Statistics:
===================
Average Request Size:     36.0 bytes
Average Response Size:    68.0 bytes
Overall Amplification:    1.89x
```

## License

Copyright 2025 by its authors. Some rights reserved. See [AUTHORS](AUTHORS).

MIT License - see [LICENSE](LICENSE) for full text.

## Acknowledgments

Code adopted from [pion/stun](https://github.com/pion/stun) and [pion/turn](https://github.com/pion/turn).
