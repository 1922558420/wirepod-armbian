package wirepod_ttr

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
)

func containsHan(input string) bool {
	return strings.ContainsFunc(input, func(r rune) bool { return unicode.Is(unicode.Han, r) })
}

func synthesizeMandarin(input string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	speech := exec.CommandContext(ctx, "espeak-ng", "-v", "cmn", "-s", "148", "--stdout", "--stdin")
	if _, err := os.Stat("/opt/wirepod-mandarin/synthesize"); err == nil {
		speech = exec.CommandContext(ctx, "/opt/wirepod-mandarin/synthesize")
	}
	speech.Stdin = strings.NewReader(input)
	wav, err := speech.Output()
	if err != nil {
		return nil, fmt.Errorf("Mandarin synthesis failed: %w", err)
	}
	convert := exec.CommandContext(ctx, "sox", "-t", "wav", "-", "-t", "raw", "-r", "16000", "-c", "1", "-b", "16", "-e", "signed-integer", "-L", "-", "gain", "-3")
	convert.Stdin = bytes.NewReader(wav)
	pcm, err := convert.Output()
	if err != nil {
		return nil, fmt.Errorf("Mandarin audio conversion failed: %w", err)
	}
	if len(pcm) == 0 || len(pcm)%2 != 0 {
		return nil, fmt.Errorf("Mandarin synthesizer returned invalid PCM audio")
	}
	return pcm, nil
}

func playMandarin(input string, robot *vector.Vector) error {
	pcm, err := synthesizeMandarin(input)
	if err != nil {
		return err
	}
	return playMandarinPCM(pcm, robot)
}

func playMandarinPCM(pcm []byte, robot *vector.Vector) error {
	duration := time.Duration(len(pcm)) * time.Second / 32000
	ctx, cancel := context.WithTimeout(context.Background(), duration+8*time.Second)
	defer cancel()
	stream, err := robot.Conn.ExternalAudioStreamPlayback(ctx)
	if err != nil {
		return err
	}
	defer stream.CloseSend()
	if err := stream.Send(&vectorpb.ExternalAudioStreamRequest{
		AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamPrepare{
			AudioStreamPrepare: &vectorpb.ExternalAudioStreamPrepare{AudioFrameRate: 16000, AudioVolume: 100},
		},
	}); err != nil {
		return err
	}
	for start := 0; start < len(pcm); start += 1024 {
		end := start + 1024
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := stream.Send(&vectorpb.ExternalAudioStreamRequest{
			AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamChunk{
				AudioStreamChunk: &vectorpb.ExternalAudioStreamChunk{AudioChunkSizeBytes: uint32(end - start), AudioChunkSamples: pcm[start:end]},
			},
		}); err != nil {
			return err
		}
		time.Sleep(30 * time.Millisecond)
	}
	if err := stream.Send(&vectorpb.ExternalAudioStreamRequest{
		AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamComplete{
			AudioStreamComplete: &vectorpb.ExternalAudioStreamComplete{},
		},
	}); err != nil {
		return err
	}
	for {
		response, err := stream.Recv()
		if err != nil {
			return err
		}
		if response.GetAudioStreamPlaybackComplete() != nil {
			logger.Println("Mandarin audio playback complete")
			return nil
		}
		if response.GetAudioStreamPlaybackFailyer() != nil || response.GetAudioStreamBufferOverrun() != nil {
			return fmt.Errorf("Vector failed to play Mandarin audio")
		}
	}
}
