package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"strconv"
	"time"
)

// -----------------------------------------------
type ImageScaleRequest struct {
	Bucket      string `json:"bucket"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ForceRemote bool   `json:"force_remote"`
	ForceBackup bool   `json:"force_backup"`
}

type ImageScaleResponse struct {
	ForceRemote      bool    `json:"force_remote"`
	CodeDuration     float64 `json:"code_duration"`
	DownloadDuration float64 `json:"download_duration"`
	ScaleDuration    float64 `json:"scale_duration"`
	UploadDuration   float64 `json:"upload_duration"`
}

type ImageRecognitionRequest struct {
	Bucket      string `json:"bucket"`
	Source      string `json:"source"`
	ForceRemote bool   `json:"force_remote"`
	ForceBackup bool   `json:"force_backup"`
}

type ImageRecognitionResponse struct {
	Predictions       []string `json:"predictions"`
	ForceRemote       bool     `json:"force_remote"`
	ShortResult       bool     `json:"short_result"`
	CodeDuration      float64  `json:"code_duration"`
	DownloadDuration  float64  `json:"download_duration"`
	InferenceDuration float64  `json:"inference_duration"`
}

type ImageScaleResult struct {
	Duration float64            `json:"duration"`
	Response ImageScaleResponse `json:"response"`
}

type ImageRecognitionResult struct {
	Duration float64                  `json:"duration"`
	Response ImageRecognitionResponse `json:"response"`
}

// ---------------------------------------
type VideoSplitRequest struct {
	Bucket      string `json:"bucket"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ForceRemote bool   `json:"force_remote"`
}
type VideoSplitResponse struct {
	ForceRemote      bool    `json:"force_remote"`
	CodeDuration     float64 `json:"code_duration"`
	DownloadDuration float64 `json:"download_duration"`
	SplitDuration    float64 `json:"split_duration"`
	UploadDuration   float64 `json:"upload_duration"`
}
type VideoSplitResult struct {
	Duration float64            `json:"duration"`
	Response VideoSplitResponse `json:"response"`
}

type VideoTranscodeRequest struct {
	Bucket      string `json:"bucket"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ForceRemote bool   `json:"force_remote"`
}
type VideoTranscodeResponse struct {
	ForceRemote       bool    `json:"force_remote"`
	CodeDuration      float64 `json:"code_duration"`
	DownloadDuration  float64 `json:"download_duration"`
	TranscodeDuration float64 `json:"transcode_duration"`
	UploadDuration    float64 `json:"upload_duration"`
}
type VideoTranscodeResult struct {
	Duration float64                `json:"duration"`
	Response VideoTranscodeResponse `json:"response"`
}

type VideoMergeRequest struct {
	Bucket      string `json:"bucket"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ForceRemote bool   `json:"force_remote"`
}
type VideoMergeResponse struct {
	ForceRemote      bool    `json:"force_remote"`
	CodeDuration     float64 `json:"code_duration"`
	DownloadDuration float64 `json:"download_duration"`
	MergeDuration    float64 `json:"merge_duration"`
	UploadDuration   float64 `json:"upload_duration"`
}
type VideoMergeResult struct {
	Duration float64            `json:"duration"`
	Response VideoMergeResponse `json:"response"`
}

// ---------------------------------------------
type FunctionChainResult struct {
	StartTs  int64                  `json:"start_ts"`
	Duration float64                `json:"duration"`
	IsResult ImageScaleResult       `json:"is_result"`
	IrResult ImageRecognitionResult `json:"ir_result"`
	VsResult VideoSplitResult       `json:"vs_result"`
	VtResult VideoTranscodeResult   `json:"vt_result"`
	VmResult VideoMergeResult       `json:"vm_result"`
}

func function_chain(index int, bucket string, source string, forceRemote bool, useMem bool, workflowType string) FunctionChainResult {
	fmt.Println("function_chain", index, "start")
	var is_result ImageScaleResult
	var ir_result ImageRecognitionResult
	var vs_result VideoSplitResult
	var vt_result VideoTranscodeResult
	var vm_result VideoMergeResult
	// ==================== function ====================
	switch workflowType {
	case "VideoProcessing":
		split_file_dst := fmt.Sprintf("%s_%d-splitted", source, index)
		merge_file_dst := fmt.Sprintf("%s_%d-transcoded", source, index)
		dst := fmt.Sprintf("%s_%d-merged", source, index)
		start := time.Now()
		vs_result = function_video_split(bucket, source, split_file_dst, forceRemote, useMem)

		var transcode_result [5]VideoTranscodeResult
		var wg sync.WaitGroup
		wg.Add(5)
		for i := 0; i < 5; i++ {
			go func(i int) {
				defer wg.Done()
				var splited_file_path = split_file_dst + "/seg" + strconv.Itoa(i+1) + "_sample.mp4"
				transcode_result[i] = function_video_transcode(bucket, splited_file_path, merge_file_dst, forceRemote, useMem)
			}(i)
		}
		wg.Wait()
		vt_result.Duration = 0.0
		vt_result.Response = VideoTranscodeResponse{
			ForceRemote:       forceRemote,
			CodeDuration:      0.0,
			DownloadDuration:  0.0,
			TranscodeDuration: 0.0,
			UploadDuration:    0.0,
		}
		for i := 0; i < 5; i++ {
			vt_result.Duration += transcode_result[i].Duration
			vt_result.Response.CodeDuration += transcode_result[i].Response.CodeDuration
			vt_result.Response.DownloadDuration += transcode_result[i].Response.DownloadDuration
			vt_result.Response.TranscodeDuration += transcode_result[i].Response.TranscodeDuration
			vt_result.Response.UploadDuration += transcode_result[i].Response.UploadDuration
		}
		vt_result.Duration /= 5
		vt_result.Response.CodeDuration /= 5
		vt_result.Response.DownloadDuration /= 5
		vt_result.Response.TranscodeDuration /= 5
		vt_result.Response.UploadDuration /= 5

		vm_result := function_video_merge(bucket, merge_file_dst, dst, forceRemote, useMem)
		duration := time.Since(start)

		fmt.Println("function_chain", index, "end")

		return FunctionChainResult{int64(start.UnixMicro()), duration.Seconds(), is_result, ir_result, vs_result, vt_result, vm_result}
	case "ImageProcessing":
		intermediate := fmt.Sprintf("%s_%d-scaled", source, index)
		start := time.Now()
		is_result := function_image_scale(bucket, source, intermediate, forceRemote, useMem)
		// time.Sleep(10 * time.Second)
		ir_result := function_image_recognition(intermediate, forceRemote, useMem)
		duration := time.Since(start)

		fmt.Println("function_chain", index, "end")

		return FunctionChainResult{int64(start.UnixMicro()), duration.Seconds(), is_result, ir_result, vs_result, vt_result, vm_result}
	default:
		fmt.Println("Unknown workflow type.")
		return FunctionChainResult{int64(0), 30, is_result, ir_result, vs_result, vt_result, vm_result}
	}

}

func function_image_scale(bucket string, source string, destination string, forceRemote bool, useMem bool) ImageScaleResult {
	var req_data ImageScaleRequest
	var res_data ImageScaleResponse

	req_data.Bucket = bucket
	req_data.Source = source
	req_data.Destination = destination
	req_data.ForceRemote = forceRemote
	req_data.ForceBackup = false

	req := new(bytes.Buffer)
	err := json.NewEncoder(req).Encode(req_data)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	url := "http://image-scale.default.127.0.0.1.sslip.io"
	if !useMem {
		url = "http://image-scale-disk.default.127.0.0.1.sslip.io"
	}
	res, err := http.Post(url, "application/json", req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	duration := time.Since(start)

	p := make([]byte, 1024)
	n, _ := res.Body.Read(p)
	err = json.Unmarshal(p[:n], &res_data)
	if err != nil {
		fmt.Println(string(p[:n]))
		// panic(err)
	}

	return ImageScaleResult{duration.Seconds(), res_data}
}

func function_image_recognition(source string, forceRemote bool, useMem bool) ImageRecognitionResult {
	var req_data ImageRecognitionRequest
	var res_data ImageRecognitionResponse

	req_data.Bucket = "stress-benchmark"
	req_data.Source = source
	req_data.ForceRemote = forceRemote
	req_data.ForceBackup = false

	req := new(bytes.Buffer)
	err := json.NewEncoder(req).Encode(req_data)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	url := "http://image-recognition.default.127.0.0.1.sslip.io"
	if !useMem {
		url = "http://image-recognition-disk.default.127.0.0.1.sslip.io"
	}
	res, err := http.Post(url, "application/json", req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	duration := time.Since(start)

	p := make([]byte, 1024)
	n, _ := res.Body.Read(p)
	err = json.Unmarshal(p[:n], &res_data)
	if err != nil {
		fmt.Println(string(p[:n]))
		// panic(err)
	}

	return ImageRecognitionResult{duration.Seconds(), res_data}
}

func function_video_split(bucket string, source string, destination string, forceRemote bool, useMem bool) VideoSplitResult {
	var req_data VideoSplitRequest
	var res_data VideoSplitResponse

	req_data.Bucket = bucket
	req_data.Source = source
	req_data.Destination = destination
	req_data.ForceRemote = forceRemote

	req := new(bytes.Buffer)
	err := json.NewEncoder(req).Encode(req_data)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	fmt.Println(req)
	url := "http://video-split.default.127.0.0.1.sslip.io"
	// if !useMem {
	// 	url = "http://image-scale-disk.default.127.0.0.1.sslip.io"
	// }
	res, err := http.Post(url, "application/json", req)
	if err != nil {
		fmt.Println("err")
		panic(err)
	}
	defer res.Body.Close()
	duration := time.Since(start)
	p := make([]byte, 1024)
	n, _ := res.Body.Read(p)
	err = json.Unmarshal(p[:n], &res_data)
	if err != nil {
		fmt.Println(string(p[:n]))
		// panic(err)
	}

	return VideoSplitResult{duration.Seconds(), res_data}
}

func function_video_transcode(bucket string, source string, destination string, forceRemote bool, useMem bool) VideoTranscodeResult {
	var req_data VideoTranscodeRequest
	var res_data VideoTranscodeResponse

	req_data.Bucket = bucket
	req_data.Source = source
	req_data.Destination = destination
	req_data.ForceRemote = forceRemote

	req := new(bytes.Buffer)
	err := json.NewEncoder(req).Encode(req_data)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	url := "http://video-transcode.default.127.0.0.1.sslip.io"
	// if !useMem {
	// 	url = "http://image-scale-disk.default.127.0.0.1.sslip.io"
	// }
	res, err := http.Post(url, "application/json", req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	duration := time.Since(start)

	p := make([]byte, 1024)
	n, _ := res.Body.Read(p)
	err = json.Unmarshal(p[:n], &res_data)
	if err != nil {
		fmt.Println(string(p[:n]))
		// panic(err)
	}

	return VideoTranscodeResult{duration.Seconds(), res_data}
}

func function_video_merge(bucket string, source string, destination string, forceRemote bool, useMem bool) VideoMergeResult {
	var req_data VideoMergeRequest
	var res_data VideoMergeResponse

	req_data.Bucket = bucket
	req_data.Source = source
	req_data.Destination = destination
	req_data.ForceRemote = forceRemote

	req := new(bytes.Buffer)
	err := json.NewEncoder(req).Encode(req_data)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	url := "http://video-merge.default.127.0.0.1.sslip.io"
	// if !useMem {
	// 	url = "http://image-scale-disk.default.127.0.0.1.sslip.io"
	// }
	res, err := http.Post(url, "application/json", req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	duration := time.Since(start)

	p := make([]byte, 1024)
	n, _ := res.Body.Read(p)
	err = json.Unmarshal(p[:n], &res_data)
	if err != nil {
		fmt.Println(string(p[:n]))
		// panic(err)
	}

	return VideoMergeResult{duration.Seconds(), res_data}
}
