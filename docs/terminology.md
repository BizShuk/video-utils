# video-utils — 術語表 (Terminology)

本檔是領域名詞、狀態值與縮寫的單一定義來源。Go API、Cobra 命令與文件使用同一組正名。

## 模組定位 (Module Positioning)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- | --- |
| 可重用命令樹 | Reusable Command Tree | 本模組`不`產生獨立執行檔；它提供一棵 Cobra 命令樹供宿主 CLI 註冊 | `cmd.VideoCmd` |
| 宿主 CLI | Host CLI | 註冊 `cmd.VideoCmd` 的外部執行檔，例如 `videoutils` | `README.md` |
| 前處理 | Preprocessing | 把媒體檔轉成下游可消費形式的階段性處理；本模組的全部職責 | 模組定位 |

## 處理階段 (Processing Stages)

| 術語 (Term) | 命令 (Command) | 定義 (Definition) | 套件 (Package) |
| --- | --- | --- | --- |
| 音軌抽取 | `audio` | 從影片抽出音軌，輸出轉錄就緒的 WAV | `audio/` |
| 白噪降噪 | `denoise` | 削減穩態白噪，輸出降噪後 WAV | `audio.ReduceWhiteNoise` |
| 影格取樣 | `frames` | 依間隔或場景變化取靜態影格 | `frames/` |
| 字幕產生 | `subtitles` | 抽音軌 → 呼叫轉錄器 → 寫出 SRT | `subtitles/` |
| 音訊切段 | `cut-audio` | 依固定時長切成多段 WAV | `segment/` |
| 影片切段 | `cut-video` | 依固定時長切成多段影片 | `segment/` |

## 參數概念 (Parameter Concepts)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 旗標 (Flag) |
| --- | --- | --- | --- |
| 來源窗口 | Source Window | 只處理原始媒體的某一時間區間 | `--from` / `--to` |
| 段長 | Segment Duration | 每一段的長度；`即使指定了來源窗口，段長仍由它決定` | `--duration` |
| 取樣間隔 | Interval | 影格取樣的時間間隔 | `--interval` |
| 降噪幅度 | Reduction dB | 削減的分貝數，預設 `12 dB` | `--reduction-db` |
| 噪底估計 | Noise Floor dB | 估計的噪音基準，預設 `-50 dB` | `--noise-floor-db` |
| 轉錄引擎 | Transcriber Engine | 可插換的語音轉文字後端，例如 whisper | `--engine` |

## 輸出契約 (Output Contract)

`這是本模組最重要的對外承諾`：

| 通道 (Stream) | 內容 (Content) |
| --- | --- |
| `stdout` | `只有路徑`。單一產物輸出一行；多個產物每行一個路徑 |
| `stderr` | 資訊性摘要與警告 |

因此呼叫端可以直接擷取或接管線：

```bash
audio_path="$(videoutils video audio input.mp4 --out output/audio.wav)"
```

任何往 `stdout` 寫入非路徑內容的改動都是 breaking change。

## 外部相依 (External Dependencies)

| 術語 (Term) | 定義 (Definition) | 檢查方式 |
| --- | --- | --- |
| `ffmpeg` | 所有轉換的實際執行者，必須在 `PATH` 上 | `ffmpegutil` 可用性檢查 |
| `ffprobe` | 媒體時長探測 | `ffmpegutil` duration helper |
| `afftdn` | FFmpeg 的降噪濾鏡；缺少它則 `denoise` 不可用 | 建置版本相關 |
| 轉錄執行時 | Whisper.cpp 或 Qwen3/MLX；`可選`，缺少時無法產生真實字幕 | `--whisper-bin` / `--whisper-model` |

## 邊界約束 (Boundary Constraint)

本模組`不` import `github.com/bizshuk/agentsdk`。
它是純粹的媒體前處理模組，不得引入 agent 或 LLM 相依。
