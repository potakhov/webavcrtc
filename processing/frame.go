package processing

import "github.com/pion/rtp"

const (
	FrameAudioOpus = iota
	FrameVideoH264
)

type FrameType int

type Frame struct {
	Type      FrameType
	Key       bool
	Timestamp uint32
	Data      []byte
}

type RTPJitter struct {
	clockrate    uint32
	cap          uint16
	packetsCount uint32
	nextSeqNum   uint16
	packets      []*rtp.Packet
	packetsSeqs  []uint16

	lastTime uint32
	nextTime uint32

	maxWaitTime uint32
	clockInMS   uint32
}

func NewJitter(cap uint16, clockrate uint32) *RTPJitter {
	jitter := &RTPJitter{}
	jitter.packets = make([]*rtp.Packet, cap)
	jitter.packetsSeqs = make([]uint16, cap)
	jitter.cap = cap
	jitter.clockrate = clockrate
	jitter.clockInMS = clockrate / 1000
	jitter.maxWaitTime = 100
	return jitter
}

func (jitter *RTPJitter) Add(packet *rtp.Packet) bool {

	idx := packet.SequenceNumber % jitter.cap
	jitter.packets[idx] = packet
	jitter.packetsSeqs[idx] = packet.SequenceNumber

	if jitter.packetsCount == 0 {
		jitter.nextSeqNum = packet.SequenceNumber - 1
		jitter.nextTime = packet.Timestamp
	}

	jitter.lastTime = packet.Timestamp
	jitter.packetsCount++
	return true
}

func (jitter *RTPJitter) SetMaxWaitTime(wait uint32) {
	jitter.maxWaitTime = wait
}

func (jitter *RTPJitter) GetOrdered() (out []*rtp.Packet) {
	nextSeq := jitter.nextSeqNum + 1
	for {
		idx := nextSeq % jitter.cap
		if jitter.packetsSeqs[idx] != nextSeq {
			// if we reach max wait time
			if (jitter.lastTime - jitter.nextTime) > jitter.maxWaitTime*jitter.clockInMS {
				nextSeq++
				continue
			}
			break
		}
		packet := jitter.packets[idx]
		if packet == nil {
			nextSeq++
			continue
		}
		jitter.packets[idx] = nil
		out = append(out, packet)
		jitter.nextTime = packet.Timestamp
		jitter.nextSeqNum = nextSeq
		nextSeq++
	}
	return
}
