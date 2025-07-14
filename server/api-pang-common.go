// =================================================================================
// Filename: api-pang-common.go
// Function: common functions for pang API
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2022-2024
// =================================================================================
package main

import (
	"fmt"
	"log"
)

// ---------------------------------------------------------------------------------
const (
	RSSP_MARK_RTXT     = "RTXT"      // text message for data, ex) mime, api path
	RSSP_MARK_RBIN     = "RBIN"      // binary message for data (media)
	RSSP_MARK_RCTL     = "RCTL"      // control commands in text message
	RSSP_MARK_REXT     = "REXT"      // extended control commands in text message
	RSSP_MARK_SIZE     = 4           // 4CC for prefix or suffix
	RSSP_MAX_TEXT_SIZE = 1024        // max text size
	RSSP_MAX_DATA_SIZE = 1024 * 1024 // max data size
)

// ---------------------------------------------------------------------------------
func GetChannelSourceTrack(channel, source, track string) (chn *Channel, src *Source, trk *Track, err error) {
	log.Println("i.GetChannelSourceTrack:", channel, source, track)

	chn = pStudio.findChannelByID(channel)
	if chn == nil {
		err = fmt.Errorf("not found channel: %s", channel)
		return
	}

	src, trk, err = chn.findSourceTrackByLabel(source, track)
	if err != nil {
		err = fmt.Errorf("not found source/track: %s/%s", source, track)
		return
	}
	return
}

//=================================================================================
