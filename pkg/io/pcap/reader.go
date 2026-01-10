// Package pcap provides PCAP file reading and network packet feature extraction.
package pcap

import (
	"context"
	"errors"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// Reader reads packets from PCAP files or live interfaces.
type Reader struct {
	handle    *pcap.Handle
	extractor *FeatureExtractor
	isLive    bool
}

// NewFileReader creates a reader for PCAP files.
func NewFileReader(filename string) (*Reader, error) {
	handle, err := pcap.OpenOffline(filename)
	if err != nil {
		return nil, err
	}

	return &Reader{
		handle:    handle,
		extractor: NewFeatureExtractor(),
		isLive:    false,
	}, nil
}

// NewLiveReader creates a reader for live packet capture.
func NewLiveReader(iface string, snaplen int32, promisc bool, timeout time.Duration) (*Reader, error) {
	handle, err := pcap.OpenLive(iface, snaplen, promisc, timeout)
	if err != nil {
		return nil, err
	}

	return &Reader{
		handle:    handle,
		extractor: NewFeatureExtractor(),
		isLive:    true,
	}, nil
}

// Read returns all packets as feature vectors.
func (r *Reader) Read() ([][]float64, error) {
	if r.handle == nil {
		return nil, errors.New("reader not initialized")
	}

	var data [][]float64
	packetSource := gopacket.NewPacketSource(r.handle, r.handle.LinkType())

	for packet := range packetSource.Packets() {
		features := r.extractor.Extract(packet)
		if features != nil {
			data = append(data, features)
		}
	}

	return data, nil
}

// Stream returns a channel of feature vectors for real-time processing.
func (r *Reader) Stream(ctx context.Context) (<-chan []float64, error) {
	if r.handle == nil {
		return nil, errors.New("reader not initialized")
	}

	out := make(chan []float64, 1000)
	packetSource := gopacket.NewPacketSource(r.handle, r.handle.LinkType())

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case packet, ok := <-packetSource.Packets():
				if !ok {
					return
				}
				features := r.extractor.Extract(packet)
				if features != nil {
					select {
					case out <- features:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return out, nil
}

// Close releases resources.
func (r *Reader) Close() error {
	if r.handle != nil {
		r.handle.Close()
	}
	return nil
}

// FeatureExtractor extracts numerical features from network packets.
type FeatureExtractor struct {
	lastTimestamp time.Time
}

// NewFeatureExtractor creates a new packet feature extractor.
func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{}
}

// Extract converts a packet to a feature vector.
// Features: [packet_size, inter_arrival_time, protocol, src_port, dst_port,
//            tcp_flags, ip_ttl, payload_size]
func (e *FeatureExtractor) Extract(packet gopacket.Packet) []float64 {
	features := make([]float64, 8)

	// Packet size
	features[0] = float64(len(packet.Data()))

	// Inter-arrival time
	metadata := packet.Metadata()
	if metadata != nil && !metadata.Timestamp.IsZero() {
		if !e.lastTimestamp.IsZero() {
			features[1] = metadata.Timestamp.Sub(e.lastTimestamp).Seconds()
		}
		e.lastTimestamp = metadata.Timestamp
	}

	// Protocol
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		features[2] = 6 // TCP
		tcp := tcpLayer.(*layers.TCP)
		features[3] = float64(tcp.SrcPort)
		features[4] = float64(tcp.DstPort)
		features[5] = encodeTCPFlags(tcp)
	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		features[2] = 17 // UDP
		udp := udpLayer.(*layers.UDP)
		features[3] = float64(udp.SrcPort)
		features[4] = float64(udp.DstPort)
	} else if packet.Layer(layers.LayerTypeICMPv4) != nil {
		features[2] = 1 // ICMP
	}

	// IP TTL
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip := ipLayer.(*layers.IPv4)
		features[6] = float64(ip.TTL)
	}

	// Payload size
	if appLayer := packet.ApplicationLayer(); appLayer != nil {
		features[7] = float64(len(appLayer.Payload()))
	}

	return features
}

// FeatureNames returns the names of extracted features.
func (e *FeatureExtractor) FeatureNames() []string {
	return []string{
		"packet_size",
		"inter_arrival_time",
		"protocol",
		"src_port",
		"dst_port",
		"tcp_flags",
		"ip_ttl",
		"payload_size",
	}
}

// encodeTCPFlags converts TCP flags to a numeric value.
func encodeTCPFlags(tcp *layers.TCP) float64 {
	var flags float64
	if tcp.SYN {
		flags += 1
	}
	if tcp.ACK {
		flags += 2
	}
	if tcp.FIN {
		flags += 4
	}
	if tcp.RST {
		flags += 8
	}
	if tcp.PSH {
		flags += 16
	}
	if tcp.URG {
		flags += 32
	}
	return flags
}
