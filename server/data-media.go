// =================================================================================
// Filename: data-media.go
// Function: Source and Track handing in Channel
// Copyright: TeamGRIT, 2021-2023
// Author: Stoney Kang, sikang@teamgrit.kr
// =================================================================================
package main

//---------------------------------------------------------------------------------
// func (d *Studio) findChannelSourceTrack(cid, source, track string) (c *Channel, err error) {
// 	c = d.findChannelByID(cid)
// 	if c == nil {
// 		err = fmt.Errorf("not found channel %s", cid)
// 		return
// 	}
// 	_, _, err = c.findSourceTrackByLabel(source, track)
// 	if err != nil {
// 		err = fmt.Errorf("invalid source %s / track %s", source, track)
// 		return
// 	}
// 	return
// }

// ---------------------------------------------------------------------------------
// check if there is no sessions on the channel and if it is, set it idle state
func (d *Studio) afterSetChannelIdleByID(id string) bool {
	if d.countSessionsByChannelID(id) == 0 {
		if d.setChannelByIDState(id, Idle) != nil {
			return true
		}
	}
	return false
}

func (d *Studio) afterSetBridgeIdleByID(id string) bool {
	if d.countSessionsByBridgeID(id) == 0 {
		if d.setBridgeByIDState(id, Idle) != nil {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------------
// check if the channel is permitted to use and
// the resource (channel/source/track) is used by any publisher
func (d *Studio) checkResourceAvailable(qo QueryOption) bool {
	if qo.Track.Style == "multi" ||
		d.findPublisherByResource(qo.Channel.ID, qo.Source.Label, qo.Track.Label) == nil {
		return true
	}
	return false
}

// func (d *Studio) checkChannelPermission(cid, key string) bool {
// 	c := d.findChannelByID(cid)
// 	return c.StreamKey == "" || c.StreamKey == key
// }

//=================================================================================
