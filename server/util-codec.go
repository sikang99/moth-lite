// =================================================================================
// Filename: util-codec.go
// Function: Codec handling
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2023
// =================================================================================
package main

import "log"

// ---------------------------------------------------------------------------------
func IsKeyFrame(codec string, data []byte) (fkey bool) {
	log.Println("i.IsKeyFrame:", codec)

	switch codec {
	case "jpeg":
		fkey = true
	case "h264":
		// 00 00 00 01 65(IDR), 67(SPS), 68(PPS)
		if data[5] == 0x65 || data[5] == 0x67 || data[5] == 0x68 {
			fkey = true
		}
	case "vp8", "vp9":
		fkey = (data[0]&0x1 == 0)
	default:
		log.Println("unknown codec:", codec)
		fkey = false
	}
	return
}

//=================================================================================
