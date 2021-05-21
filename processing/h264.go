package processing

import (
	"log"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/potakhov/webavcrtc/codec/h264"
)

type H264StreamCallback func(frame *Frame)

type H264RTPDepacketizer struct {
	frame         []byte
	timestamp     uint32
	onframe       H264StreamCallback
	h264Unmarshal *codecs.H264Packet
	jitter        *RTPJitter
}

func NewH264Depacketizer() *H264RTPDepacketizer {
	return &H264RTPDepacketizer{
		frame:         make([]byte, 0),
		h264Unmarshal: &codecs.H264Packet{},
		jitter:        NewJitter(512, 90000),
	}
}

func (dpkt *H264RTPDepacketizer) OnFrame(cb H264StreamCallback) {
	dpkt.onframe = cb
}

func parseConvertNALu(frame []byte) ([]byte, bool) {
	nalus, _ := h264.SplitNALUs(frame)

	nalus_ := make([][]byte, 0)
	keyframe := false

	var stmp string
	for _, nalu := range nalus {

		switch h264.NALUType(nalu) {
		case h264.NALU_SPS:
			nalus_ = append(nalus_, nalu)
			stmp += "SPS "
		case h264.NALU_PPS:
			nalus_ = append(nalus_, nalu)
			stmp += "PPS "
		case h264.NALU_IDR:
			nalus_ = append(nalus_, nalu)
			keyframe = true
			stmp += "IDR "
		case h264.NALU_NONIDR:
			nalus_ = append(nalus_, nalu)
			keyframe = false
			stmp += "NONIDR "
		case h264.NALU_SEI:
			stmp += "SEI "
			continue
		default:
			stmp += "UNK "
			continue
		}
	}

	log.Printf("Detected NALs: %s", stmp)

	if len(nalus_) == 0 {
		return nil, false
	}

	return h264.FillNALUsAnnexb(nalus_), keyframe
}

func (dpkt *H264RTPDepacketizer) AddPacket(packet *rtp.Packet) {
	dpkt.jitter.Add(packet)

	pkts := dpkt.jitter.GetOrdered()
	for _, pkt := range pkts {
		ts := pkt.Timestamp

		if dpkt.timestamp != ts {
			dpkt.frame = make([]byte, 0)
		}

		dpkt.timestamp = ts

		buf, _ := dpkt.h264Unmarshal.Unmarshal(pkt.Payload)

		dpkt.frame = append(dpkt.frame, buf...)

		if !pkt.Marker {
			continue
		}

		if dpkt.onframe != nil {
			fr, key := parseConvertNALu(dpkt.frame)
			if fr != nil {
				dpkt.onframe(&Frame{
					Type:      FrameVideoH264,
					Key:       key,
					Data:      fr,
					Timestamp: dpkt.timestamp,
				})
			}
		}
	}
}
