//go:build !windows && !cgo

package vad

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"
)

// GetVADInfoFromAudio 在非 Windows 且无法使用 cgo 的构建场景下，回退到轻量能量阈值实现。
func GetVADInfoFromAudio(audioInfo AudioInfo, insert bool) ([]VADInfo, error) {
	var (
		frameIndex  = 0
		frameBuffer = make([]byte, audioInfo.SampleRate/1000*FrameDuration*audioInfo.BitDepth/8)
		frameActive = false
		vadInfos    = make([]VADInfo, 0)
	)

	audioFile, err := os.Open(audioInfo.FileFullPath)
	if err != nil {
		return nil, err
	}
	defer audioFile.Close()

	if audioInfo.BitDepth%8 != 0 || audioInfo.BitDepth <= 0 {
		return nil, fmt.Errorf("unsupported bit depth %d", audioInfo.BitDepth)
	}

	reader := bufio.NewReader(audioFile)
	var (
		offset     int
		noiseFloor = 0.01
	)

	report := func() {
		t := time.Duration(offset) * time.Second / time.Duration(audioInfo.SampleRate) / 2
		vadInfos = append(vadInfos, *NewVADInfo(
			frameIndex,
			offset,
			frameActive,
			t,
		))
	}

	for {
		_, err = io.ReadFull(reader, frameBuffer)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}

		frameEnergy, err := frameEnergyLevel(frameBuffer, audioInfo.BitDepth)
		if err != nil {
			return nil, err
		}
		threshold := math.Max(noiseFloor*2.5, 0.02)
		tmpFrameActive := frameEnergy >= threshold

		if tmpFrameActive == false {
			noiseFloor = noiseFloor*0.95 + frameEnergy*0.05
		}

		if tmpFrameActive != frameActive || offset == 0 {
			frameActive = tmpFrameActive
			if insert == false {
				report()
			}
		}
		if insert == true {
			report()
		}
		offset += len(frameBuffer)
		frameIndex++
	}

	report()

	return vadInfos, nil
}

func frameEnergyLevel(frameBuffer []byte, bitDepth int) (float64, error) {
	switch bitDepth {
	case 16:
		return pcm16Energy(frameBuffer), nil
	case 8:
		return pcm8Energy(frameBuffer), nil
	default:
		return 0, errors.New(fmt.Sprintf("unsupported bit depth %d", bitDepth))
	}
}

func pcm16Energy(frameBuffer []byte) float64 {
	if len(frameBuffer) < 2 {
		return 0
	}
	var (
		sum     float64
		samples int
	)
	for i := 0; i+1 < len(frameBuffer); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(frameBuffer[i : i+2]))
		normalized := float64(sample) / 32768.0
		sum += normalized * normalized
		samples++
	}
	if samples == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(samples))
}

func pcm8Energy(frameBuffer []byte) float64 {
	if len(frameBuffer) == 0 {
		return 0
	}
	var sum float64
	for _, sample := range frameBuffer {
		normalized := (float64(sample) - 128.0) / 128.0
		sum += normalized * normalized
	}
	return math.Sqrt(sum / float64(len(frameBuffer)))
}

func GetFloatSlice(inVADs []VADInfo) []float64 {
	outVADFloats := make([]float64, len(inVADs))
	for i, vad := range inVADs {
		if vad.Active == true {
			outVADFloats[i] = 1
		} else {
			outVADFloats[i] = -1
		}
	}

	return outVADFloats
}

func GetAudioIndex2Time(index int) float64 {
	return float64(index*FrameDuration) / 1000.0
}

const (
	Mode          = 3
	FrameDuration = 10
)
