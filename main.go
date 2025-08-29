package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/stun/v3"
)

const stunHeaderSize = 20

type Protocol byte

const (
	// ProtoTCP is IANA assigned protocol number for TCP.
	ProtoTCP Protocol = 6
	// ProtoUDP is IANA assigned protocol number for UDP.
	ProtoUDP Protocol = 17
)

type RequestedTransport struct {
	Protocol Protocol
}

const requestedTransportSize = 4

// AddTo adds REQUESTED-TRANSPORT to message.
func (t RequestedTransport) AddTo(m *stun.Message) error {
	v := make([]byte, requestedTransportSize)
	v[0] = byte(t.Protocol)
	// b[1:4] is RFFU = 0.
	// The RFFU field MUST be set to zero on transmission and MUST be
	// ignored on reception. It is reserved for future uses.
	m.Add(stun.AttrRequestedTransport, v)

	return nil
}

type AmplificationResult struct {
	RequestSize         uint32
	ResponseSize        uint32
	AmplificationFactor float64
	ResponseType        string
	HasNonce            bool
	NonceSize           int
}

func main() {
	var (
		server = flag.String("server", "127.0.0.1:3478", "TURN server address")
		count  = flag.Int("count", 100, "Number of requests to send")
	)
	flag.Parse()

	fmt.Printf("TURN Amplification Factor Measurement Tool\n")
	fmt.Printf("==========================================\n")
	fmt.Printf("Target server: %s\n", *server)
	fmt.Printf("Request count: %d\n", *count)

	results, err := measureAmplificationFactor(*server, *count)
	if err != nil {
		log.Fatalf("Failed to measure amplification factor: %v", err)
	}

	printResults(results)
}

func measureAmplificationFactor(serverAddr string, count int) ([]AmplificationResult, error) {
	// Resolve server address
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	// Create UDP connection
	// TURN client won't create a local listening socket by itself.
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		log.Panicf("Failed to listen: %s", err)
	}
	defer conn.Close()

	var results []AmplificationResult

	for i := 0; i < count; i++ {
		result, err := sendAllocateRequest(conn, addr)
		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i+1, err)
			continue
		}

		results = append(results, result)

		time.Sleep(10 * time.Millisecond)
	}

	return results, nil
}

func sendAllocateRequest(conn net.PacketConn, to net.Addr) (AmplificationResult, error) {
	var result AmplificationResult

	// Build allocation request message
	msg, err := stun.Build(
		stun.TransactionID,
		stun.NewType(stun.MethodAllocate, stun.ClassRequest),
		RequestedTransport{Protocol: ProtoUDP},
		stun.Fingerprint,
	)
	if err != nil {
		return result, fmt.Errorf("failed to build request: %w", err)
	}

	result.RequestSize = stunHeaderSize + msg.Length

	_, err = conn.WriteTo(msg.Raw, to)
	if err != nil {
		return AmplificationResult{}, err
	}

	//
	res := make([]byte, 2048)
	if _, _, err := conn.ReadFrom(res); err != nil {
		return AmplificationResult{}, err
	}

	// Check for NONCE attribute
	msg = stun.New()
	if err := stun.Decode(res, msg); err != nil {
		return AmplificationResult{}, err
	}

	if msg.Type.Class != stun.ClassErrorResponse || msg.Type.Method != stun.MethodAllocate {
		return AmplificationResult{}, errors.New("unexpected response")
	}

	result.ResponseSize = stunHeaderSize + msg.Length
	result.AmplificationFactor = float64(result.ResponseSize) / float64(result.RequestSize)

	var nonce stun.Nonce
	if err := nonce.GetFrom(msg); err == nil {
		result.HasNonce = true
		result.NonceSize = len(nonce)
	}

	return result, nil
}

func getErrorCode(msg *stun.Message) stun.ErrorCode {
	var errorCode stun.ErrorCodeAttribute
	if err := errorCode.GetFrom(msg); err == nil {
		return errorCode.Code
	}
	return 0
}

func printResults(results []AmplificationResult) {
	if len(results) == 0 {
		fmt.Println("No successful results to analyze.")
		return
	}

	fmt.Printf("\nResults Summary\n")
	fmt.Printf("===============\n")
	fmt.Printf("Successful requests: %d\n\n", len(results))

	// Print detailed analysis for each response type
	// Overall statistics
	var totalAmp float64
	var totalReqSize, totalRespSize uint32
	for _, res := range results {
		totalAmp += res.AmplificationFactor
		totalReqSize += res.RequestSize
		totalRespSize += res.ResponseSize
	}

	avgAmp := totalAmp / float64(len(results))
	avgReqSize := float64(totalReqSize) / float64(len(results))
	avgRespSize := float64(totalRespSize) / float64(len(results))

	fmt.Printf("Overall Statistics:\n")
	fmt.Printf("===================\n")
	fmt.Printf("Average Request Size:     %.1f bytes\n", avgReqSize)
	fmt.Printf("Average Response Size:    %.1f bytes\n", avgRespSize)
	fmt.Printf("Overall Amplification:    %.2fx\n", avgAmp)
}
