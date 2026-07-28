package service

import (
	"testing"
)

func TestExtractTikHubVideoInfo(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantVideo string
		wantCover string
		wantDesc  string
		wantAuthor string
	}{
		{
			name: "标准 aweme_detail 结构",
			body: `{
				"data": {
					"aweme_detail": {
						"desc": "hello world",
						"author": {
							"unique_id": "@testuser",
							"nickname": "Test User"
						},
						"video": {
							"cover": {
								"url": "https://cover.example.com/cover.jpg"
							},
							"play_addr": {
								"url_list": ["https://play.example.com/video.mp4"]
							},
							"download_addr": {
								"url": "https://download.example.com/video.mp4"
							}
						}
					}
				}
			}`,
			wantVideo:  "https://download.example.com/video.mp4",
			wantCover:  "https://cover.example.com/cover.jpg",
			wantDesc:   "hello world",
			wantAuthor: "@testuser",
		},
		{
			name: "无水印地址优先",
			body: `{
				"data": {
					"aweme_detail": {
						"desc": "no watermark",
						"author": {"unique_id": "@nowm"},
						"video": {
							"cover": {"url": "https://cover.example.com/cover.jpg"},
							"play_addr": {"url_list": ["https://play.example.com/video.mp4"]},
							"download_addr": {"url": "https://download.example.com/video.mp4"},
							"download_no_watermark_addr": {"url_list": ["https://nowm.example.com/video.mp4"]}
						}
					}
				}
			}`,
			wantVideo:  "https://nowm.example.com/video.mp4",
			wantCover:  "https://cover.example.com/cover.jpg",
			wantDesc:   "no watermark",
			wantAuthor: "@nowm",
		},
		{
			name: "顶层 video 结构",
			body: `{
				"data": {
					"desc": "simple video",
					"author": {"nickname": "Nick"},
					"video": {
						"cover": {"url_list": ["https://cover2.example.com/cover.jpg"]},
						"play_addr": {"url": "https://play2.example.com/video.mp4"}
					}
				}
			}`,
			wantVideo:  "https://play2.example.com/video.mp4",
			wantCover:  "https://cover2.example.com/cover.jpg",
			wantDesc:   "simple video",
			wantAuthor: "Nick",
		},
		{
			name: "video_data 结构",
			body: `{
				"data": {
					"video_data": {
						"play_addr": {"url": "https://video_data.example.com/video.mp4"},
						"cover": {"url": "https://cover3.example.com/cover.jpg"}
					}
				}
			}`,
			wantVideo:  "https://video_data.example.com/video.mp4",
			wantCover:  "https://cover3.example.com/cover.jpg",
			wantDesc:   "",
			wantAuthor: "",
		},
		{
			name: "顶层 video_url 字段",
			body: `{
				"data": {
					"video_url": "https://top_level.example.com/video.mp4",
					"desc": "top level"
				}
			}`,
			wantVideo:  "https://top_level.example.com/video.mp4",
			wantCover:  "",
			wantDesc:   "top level",
			wantAuthor: "",
		},
		{
			name: "无 data 包装",
			body: `{
				"aweme_detail": {
					"desc": "no wrapper",
					"video": {
						"play_addr": {"url_list": ["https://nowrapper.example.com/video.mp4"]}
					}
				}
			}`,
			wantVideo:  "https://nowrapper.example.com/video.mp4",
			wantCover:  "",
			wantDesc:   "no wrapper",
			wantAuthor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ExtractTikHubVideoInfo([]byte(tt.body))
			if err != nil {
				t.Fatalf("ExtractTikHubVideoInfo() error = %v", err)
			}
			if info.VideoURL != tt.wantVideo {
				t.Errorf("VideoURL = %q, want %q", info.VideoURL, tt.wantVideo)
			}
			if info.CoverURL != tt.wantCover {
				t.Errorf("CoverURL = %q, want %q", info.CoverURL, tt.wantCover)
			}
			if info.Desc != tt.wantDesc {
				t.Errorf("Desc = %q, want %q", info.Desc, tt.wantDesc)
			}
			if info.Author != tt.wantAuthor {
				t.Errorf("Author = %q, want %q", info.Author, tt.wantAuthor)
			}
		})
	}
}
