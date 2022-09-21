package rtc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/livekit/livekit-server/pkg/rtc/types"
	"github.com/livekit/livekit-server/pkg/telemetry"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
)

type EgressLauncher interface {
	StartEgress(context.Context, *livekit.StartEgressRequest) (*livekit.EgressInfo, error)
}

func StartTrackEgress(
	ctx context.Context,
	launcher EgressLauncher,
	ts telemetry.TelemetryService,
	opts *livekit.AutoTrackEgress,
	track types.MediaTrack,
	roomName livekit.RoomName,
	roomID livekit.RoomID,
) {
	if req, err := startTrackEgress(ctx, launcher, opts, track, roomName, roomID); err != nil {
		// send egress failed webhook
		ts.NotifyEvent(ctx, &livekit.WebhookEvent{
			Event: webhook.EventEgressEnded,
			EgressInfo: &livekit.EgressInfo{
				RoomId:   string(roomID),
				RoomName: string(roomName),
				Status:   livekit.EgressStatus_EGRESS_FAILED,
				Error:    err.Error(),
				Request:  &livekit.EgressInfo_Track{Track: req},
			},
		})
	}
}

func startTrackEgress(
	ctx context.Context,
	launcher EgressLauncher,
	opts *livekit.AutoTrackEgress,
	track types.MediaTrack,
	roomName livekit.RoomName,
	roomID livekit.RoomID,
) (*livekit.TrackEgressRequest, error) {

	output := &livekit.DirectFileOutput{
		Filepath: getFilePath(opts.FilePrefix, string(track.ID())),
	}

	switch out := opts.Output.(type) {
	case *livekit.AutoTrackEgress_Azure:
		output.Output = &livekit.DirectFileOutput_Azure{Azure: out.Azure}
	case *livekit.AutoTrackEgress_Gcp:
		output.Output = &livekit.DirectFileOutput_Gcp{Gcp: out.Gcp}
	case *livekit.AutoTrackEgress_S3:
		output.Output = &livekit.DirectFileOutput_S3{S3: out.S3}
	}

	req := &livekit.TrackEgressRequest{
		RoomName: string(roomName),
		TrackId:  string(track.ID()),
		Output: &livekit.TrackEgressRequest_File{
			File: output,
		},
	}

	if launcher == nil {
		return req, errors.New("egress launcher not found")
	}

	_, err := launcher.StartEgress(ctx, &livekit.StartEgressRequest{
		Request: &livekit.StartEgressRequest_Track{
			Track: req,
		},
		RoomId: string(roomID),
	})
	return req, err
}

func getFilePath(prefix, identifier string) string {
	if prefix == "" || strings.HasSuffix(prefix, "/") {
		return fmt.Sprintf("%s%s_%s", prefix, identifier, time.Now().Format("2006-01-02T150405"))
	} else {
		return fmt.Sprintf("%s_%s_%s", prefix, identifier, time.Now().Format("2006-01-02T150405"))
	}
}