// =================================================================================
// Filename: internal-msg.go
// Function: handle internal command delivered via stream
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import "log"

// ---------------------------------------------------------------------------------
const (
	// Internal data
	MIME_SPIDER_JSON = "spider/json" // Spider server internal control
	MIME_HORNET_JSON = "hornet/json" // Moth server internal control
	MIME_MOTH_JSON   = "moth/json"   // Moth server internal control
	// Link protocol data
	MIME_COLINK_JSON = "colink/json"    // cobot link protocol
	MIME_MAVLINK_BIN = "mavlink/binary" // mavlink link protocol
	// Audio data
	MIME_AUDIO_WAVE = "audio/x-wav;codec=pcm" // "audio/wave"    // wave (pcm, adpcm)
	MIME_AUDIO_OPUS = "audio/opus"            // opus
	MIME_AUDIO_LYRA = "audio/lyra"            // lyra
	MIME_AUDIO_SSTM = "audio/sstream"         // sound stream
	// Video data
	MIME_VIDEO_PCC  = "video/pcc"  // video-based point cloud compression, MPEG-3DG-V-PCC
	MIME_VIDEO_JPEG = "video/jpeg" // JPEG video (MJPEG)
	MIME_VIDEO_PNG  = "video/png"  // JPEG video (MJPEG)
	MIME_VIDEO_WEBP = "video/webp" // WebP video
	MIME_VIDEO_HEIF = "video/heif" // HEIF video
	MIME_VIDEO_AVIF = "video/avif" // AVIF video
	MIME_VIDEO_H264 = "video/h264" // H.264 (AVC) video
	MIME_VIDEO_H265 = "video/h265" // H.265 (HEVC) video
	// Image data
	MIME_IMAGE_JPEG = "image/jpeg"    // single jpeg image
	MIME_IMAGE_PNG  = "image/png"     // single png image
	MIME_IMAGE_SVG  = "image/svg+xml" // single svg image
	MIME_IMAGE_WEBP = "image/webp"    // single webp image
	MIME_IMAGE_HEIF = "image/heif"    // single heif image
	MIME_IMAGE_AVIF = "image/avif"    // single avif image
	// Unknown data
	MIME_STREAM_UNKNOWN = "application/octet-stream"
)

// ---------------------------------------------------------------------------------
type IMessage struct {
	Seq    int    `json:"seq,omitempty"`
	Action string `json:"action,omitempty"`
}

// ---------------------------------------------------------------------------------
func ProcInternalJSONMessage(data []byte) (err error) {
	log.Println("i.ProcInternalMessage:")

	// TBD
	log.Println(string(data))
	return
}

//=================================================================================
