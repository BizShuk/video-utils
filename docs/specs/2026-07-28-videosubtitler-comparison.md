# VideoSubtitler 與 video-utils 比較報告

- 日期：2026-07-28
- 本地基準：`video-utils` commit
  `7bf46f267ac138c0cc993a2846030038b5de92b5`
- 上游基準：[GoldYuBrain/VideoSubtitler][upstream-repo] commit
  [`e83bba1ecf0f92cf23cd03b7361631982056d27a`][upstream-commit]
- 比較目的：判斷上游哪些產品能力值得納入 `video-utils`，以及哪些實作方式
  不應直接移植

## 結論

`VideoSubtitler` 與 `video-utils` 解決的是同一條影音字幕流程中的不同層次。

- `VideoSubtitler` 是 Windows/Python 終端使用者工具，重點是一次完成
  「語音辨識 → 繁中翻譯 → 硬字幕影片」。
- `video-utils` 是 Go library 加可重用 Cobra command tree，重點是提供可組合的
  音訊、影格及字幕前處理能力，讓 `auth-cli` 等宿主程式自行編排。

因此，不建議以 `VideoSubtitler` 取代 `video-utils`，也不建議把其 Python
單體流程直接搬進 Go module。建議保留現有 package 邊界與 path-only stdout
契約，再分階段補上：

1. 原文 `.txt` 輸出。
2. 可替換的字幕翻譯介面與繁中正規化。
3. 跨平台字幕燒錄 package。
4. 可斷點重跑、但具嚴格步驟驗證的 pipeline orchestrator。

上游最值得採用的是完整的使用者結果、可編輯中間產物及從指定階段重跑；
最不應照搬的是 Windows 預設值、隱含的網路翻譯 fallback、硬編碼模型、
副檔名式媒體判定與未驗證的任意 `--steps` 字串。

## 比較範圍與方法

本報告檢查了兩邊的 README、入口程式、字幕/音訊/FFmpeg 實作、依賴宣告及
測試狀態。上游分析固定在上述 commit，避免日後 `main` 變動造成結論漂移。

本次驗證：

- `video-utils`：`go test ./... -count=1 -timeout=120s` 通過。
- `video-utils`：`go build ./...` 通過。
- `VideoSubtitler`：所有 Python source 通過 `python3 -m compileall` 語法檢查。
- 未執行上游的完整辨識、翻譯或燒錄流程；該流程需要另行下載 Whisper
  模型、安裝 Python dependencies，並可能呼叫外部翻譯服務。

## 能力矩陣

| 面向 | `video-utils` | `VideoSubtitler` | 判斷 |
| --- | --- | --- | --- |
| 產品定位 | 可嵌入 Go module 與 Cobra stages | 直接執行的 Python CLI | 分屬基礎層與應用層 |
| 音訊抽取 | FFmpeg 輸出 16 kHz mono PCM WAV，可調 sample rate/channels | `faster-whisper` 直接讀媒體 | 本地介面較清楚、可重用 |
| 降噪 | `afftdn` 白噪音降低，有 API、CLI 與測試 | 無 | 本地獨有 |
| 影格抽取 | 固定間隔、scene change、上限、JPEG/PNG、時間戳 | 無 | 本地獨有 |
| ASR backend | `Transcriber` interface；Whisper.cpp、Qwen3/MLX、noop | `faster-whisper`，device/compute 自動選擇 | 本地可替換性較高；上游開箱體驗較好 |
| 字幕輸出 | SRT | SRT 與純文字 TXT | 應補 TXT |
| 翻譯 | 無 | Google、Claude、OpenCC，輸出 `zh-TW` SRT/TXT | 上游核心差異 |
| 字幕燒錄 | 無 | FFmpeg 硬字幕，支援字型、大小、顏色、位置、粗體 | 上游核心差異 |
| 純音訊成片 | 無 | 產生 1280×720 深色背景 MP4 | 可選用的產品能力 |
| 中間產物重用 | stages 可分開呼叫，但無統一 pipeline | `--steps` 可跳過已完成階段 | 上游 UX 值得吸收 |
| 自動化輸出 | stdout 僅輸出結果路徑；資訊走 stderr | 進度、命令、結果混在 stdout | 本地契約較適合 script/agent |
| 平台 | Go/FFmpeg 主流程可跨平台；Qwen3/MLX 限 Apple Silicon | 僅 Windows 驗證，預設微軟正黑體 | 本地基礎較跨平台 |
| 測試 | 多個 Go package 測試，含合成媒體與 FFmpeg 行為 | 無 automated tests 或 CI | 本地成熟度較高 |
| 發佈形態 | 獨立 Go module，但不產生 standalone binary | clone 後建立 venv 執行，未 package 化 | 兩邊都尚有發佈缺口 |
| 授權 | repo 目前沒有 `LICENSE` | MIT，並說明 FFmpeg 授權 | 本地應先補明確授權 |

## 架構與資料流

### `video-utils`

```text
media
├── audio.Extract ───────────────> WAV
├── audio.ReduceWhiteNoise ──────> denoised WAV
├── frames.Extract ──────────────> timestamped JPEG/PNG[]
└── subtitles.Generate
    ├── audio.Extract
    └── Transcriber
        ├── WhisperCPPTranscriber
        └── QwenMLXTranscriber
                               ──> Segment[] ──> SRT
```

核心設計是 [`subtitles.Transcriber`](../../subtitles/subtitles.go) interface。
ASR backend 只負責把 WAV 轉為 timed segments，音訊抽取、SRT 寫入與命令列輸出
互不綁定。這使新增 API 型 transcriber、測試 stub 或其他本地模型時，不必改動
主流程。

[`cmd.VideoCmd`](../../cmd/video.go) 提供 `audio`、`denoise`、`frames` 與
`subtitles` stage。成功時 stdout 只含路徑，適合 shell pipeline、agent tool
或宿主 CLI 消費。

### `VideoSubtitler`

```text
media
  └── faster-whisper
        ├── original SRT
        └── original TXT
              └── Google | Claude | OpenCC
                    ├── zh-TW SRT
                    └── zh-TW TXT
                          └── FFmpeg subtitles filter
                                └── hard-subtitled MP4
```

[`main.py`][upstream-main] 依固定順序檢查並執行三個 stage。
[`transcribe.py`][upstream-transcribe] 使用 `faster-whisper`、beam size 5 與
VAD；[`translate.py`][upstream-translate] 每 40 段批次翻譯；
[`burn.py`][upstream-burn] 用 FFmpeg `subtitles` filter 產生 H.264 MP4。

這個結構簡單、容易理解，也直接產出使用者真正需要的影片；代價是入口、
命名規則、翻譯策略與燒錄策略耦合在同一個應用流程，沒有穩定 library
contract。

## `VideoSubtitler` 值得吸收的設計

### 1. 對「完成結果」負責

本地目前把責任停在原文 SRT，使用者仍要自行找翻譯及燒錄工具。上游把
`demo.srt`、`demo.txt`、`demo.zh-TW.srt`、`demo.zh-TW.txt` 和
`demo.zh-TW.mp4` 視為同一工作成果，產品敘事更完整。

建議 `video-utils` 增加能力，但不要讓底層 packages 強制執行全流程。由新的
pipeline package 或宿主 command 決定是否串接。

### 2. 中間產物可人工修正後續跑

上游允許使用者修改原文或翻譯後 SRT，再只執行 `translate,burn` 或 `burn`。
這比「每次一鍵重跑全部」更符合字幕工作的真實流程，因為 ASR 與翻譯必然需要
人工校對。

本地應把中間產物視為正式 artifact，而不是 scratch file：

- 每個 stage 接收明確 input path 並回傳明確 output path。
- pipeline 在覆寫前顯示將使用的既有 artifact。
- 後續可加入 manifest，記錄來源檔 hash、engine、model、language 與參數，
  防止誤用其他影片的舊字幕。

### 3. SRT 與 TXT 同時輸出

純文字 transcript 適合摘要、全文搜尋、LLM ingestion 與人工閱讀。由現有
`Segment[]` 產生 TXT 成本很低，且不需要引入上游任何 Python dependency。

### 4. 可調的燒錄樣式

字型、大小、顏色、位置及粗體是合理的最小硬字幕介面。未來若加入 `burn`
package，應沿用這組產品概念，但改採跨平台預設字型策略與嚴格的顏色/數值驗證。

### 5. 純音訊可轉成字幕影片

為 podcast 或語音 memo 產生背景影片，能把輸出直接投放到影片平台。這適合放在
應用層的可選 stage，不應成為音訊或字幕 library 的隱含副作用。

## 不應直接移植的部分

### 1. `--steps` 缺乏嚴格驗證

上游把輸入以逗號拆開後只做 membership check。未知名稱會被忽略；若所有名稱
都錯，程式仍印出完成訊息。執行順序也固定為 transcribe、translate、burn，
不是使用者輸入的順序。

本地應以明確 subcommands 或 enum 驗證 stage，拒絕未知值，並在執行前驗證
dependency graph。

### 2. 翻譯 fallback 可能靜默保留原文

Google batch 失敗後會逐句重試；空翻譯會以來源文字替代。Claude 回覆若缺少某個
編號，也會以來源文字替代。這能讓流程繼續，但輸出檔看起來成功時可能仍混有
未翻譯片段。

建議 Translator 回傳每段狀態，對缺段、空值、段數不符提供：

- strict mode：整批失敗，不產生宣稱完整的譯文。
- best-effort mode：輸出檔可建立，但 stderr 與 manifest 必須列出未翻譯段落。

### 3. `auto` 隱含外部網路行為

上游未設定 `ANTHROPIC_API_KEY` 時會自動使用 Google 翻譯。這對桌面工具方便，
但不適合 library 或 agent automation：使用者可能不知道字幕內容被送到第三方，
也可能遇到非正式介面的 rate limit 或行為變更。

本地應要求明確選擇 translator；若提供 `auto`，至少先輸出 provider、
privacy/cost 提示，且不得把 API key 寫入 log。

### 4. Windows 與硬編碼執行環境

上游預設 `Microsoft JhengHei`，優先尋找
`tools/ffmpeg*/bin/ffmpeg.exe`，純音訊判定只看副檔名。這些做法適合其 Windows
目標，但不適合跨平台 module。

本地應：

- 以 `ffprobe` stream metadata 判斷是否有 video/audio stream。
- 讓宿主明確配置字型，或依平台解析可用的 CJK font。
- 保留 `PATH` lookup，不把 Windows bundle layout 變成 library contract。

### 5. 相依性與模型版本可重現性不足

上游 `requirements.txt` 全部只有最低版本，Claude model name 直接硬編碼在
source，沒有 lock file。相同命令日後可能取得不同 dependency 或模型行為。

本地新增 Python/provider integration 時，應把 model 當作設定，並提供
可重現的 dependency/版本安裝方式。

## `video-utils` 自身應先處理的缺口

比較也顯示本地並非已可直接交付給終端使用者：

1. `subtitles` command 預設使用 `noop`，命令成功但產生 0 段 SRT。雖然 stderr
   有警告，仍容易被自動化誤判為有效字幕。
2. Qwen wrapper 的 command 預設路徑是
   `utils/video/subtitles/pyasr/qwen_transcribe.py`，這是原宿主 workspace
   layout，不是獨立 `video-utils` repo layout。作為獨立 Go module 發佈時，
   資產定位方式尚未定義。
3. Qwen/MLX 的 Python dependency 只寫在註解，沒有可執行的安裝或 runtime
   preflight。
4. module 宣稱可獨立 version，但 repo 目前沒有 `LICENSE`。
5. 沒有 standalone executable；第一次使用者必須先理解如何由宿主註冊
   `cmd.VideoCmd`。
6. `ffmpegutil` 有清楚的錯誤契約，但目前沒有自己的 unit tests。

這些項目應優先於擴大功能面，否則加入翻譯與燒錄只會放大安裝和成功判定的不確定性。

## 建議目標設計

```text
audio/          WAV extraction and denoise
frames/         interval and scene sampling
subtitles/      Segment, Transcriber, SRT/TXT codecs
translate/      Translator interface and translation result status
burn/           subtitle rendering and media stream handling
ffmpegutil/     binary lookup, probe, filter escaping
pipeline/       explicit stage graph, artifact manifest, resume policy
cmd/            thin Cobra adapters; path-only stdout
```

建議介面方向：

```go
type Translator interface {
    Translate(ctx context.Context, segments []subtitles.Segment) ([]Result, error)
}

type Result struct {
    Source subtitles.Segment
    Text   string
    Status Status
}
```

`translate` 不應依賴 `cmd` 或 `burn`；`burn` 應接受既有字幕檔，不應自行觸發
翻譯；`pipeline` 才負責依序組合。這延續現有 one-way dependencies，也讓宿主
可以只用其中一個 stage。

## 建議落地順序

### P0：修正發佈與成功語意

- 決定並加入 `LICENSE`。
- 取消預設 noop 成功路徑；沒有真實 engine 時應明確失敗，或要求使用者顯式
  選 `noop`。
- 定義 Qwen wrapper 的分發與定位方式，補 runtime preflight。
- 為 `ffmpegutil` 補 binary lookup、probe parse 與錯誤測試。

### P1：低風險補齊 artifact

- 由 `Segment[]` 同時寫 SRT 與 TXT。
- 定義 artifact naming，避免相同 stem 的不同來源互相覆寫。
- 加入 metadata/manifest，支援安全 resume。

### P2：翻譯

- 建立 `Translator` interface、strict/best-effort policy 與段數驗證。
- 先支援不出網的繁簡正規化，再由宿主決定是否加入 Claude 或其他網路 provider。
- provider、model、language、成本/隱私提示全部顯式化。

### P3：燒錄與應用層 pipeline

- 新增跨平台 FFmpeg burn package，以 ffprobe 判斷 stream。
- 支援硬字幕樣式；軟字幕可作獨立後續能力。
- 在宿主 CLI 加入完整 pipeline 與可重跑 stages。
- 純音訊背景成片維持 opt-in。

## 最終採用判斷

| 決策 | 項目 |
| --- | --- |
| 直接採用概念 | TXT artifact、人工校正後重跑、字幕樣式、純音訊成片 |
| 重新設計後採用 | Translator abstraction、繁中正規化、pipeline resume、硬字幕 |
| 保留本地既有設計 | Go package 邊界、pluggable Transcriber、FFmpeg preprocessing、path-only stdout |
| 不採用 | Windows 專屬預設、隱含 Google fallback、硬編碼 Claude model、副檔名媒體判定、寬鬆 `--steps` |

整體建議是「吸收產品流程，不搬運應用程式」。先把 `video-utils` 的獨立發佈、
runtime discovery 與成功語意補完整，再逐層擴充翻譯和燒錄能力。

## 來源

- [GoldYuBrain/VideoSubtitler repository][upstream-repo]
- [上游 README（固定 commit）][upstream-readme]
- [上游 CLI orchestration][upstream-main]
- [上游 faster-whisper transcription][upstream-transcribe]
- [上游 translation engines][upstream-translate]
- [上游 FFmpeg subtitle burning][upstream-burn]
- [上游 FFmpeg lookup and media detection][upstream-ffmpeg]
- 本地 [`README.md`](../../README.md)
- 本地 [`CLAUDE.md`](../../CLAUDE.md)
- 本地 [`audio`](../../audio)、[`frames`](../../frames)、
  [`subtitles`](../../subtitles)、[`cmd`](../../cmd) 及
  [`ffmpegutil`](../../ffmpegutil)

[upstream-repo]: https://github.com/GoldYuBrain/VideoSubtitler
[upstream-commit]: https://github.com/GoldYuBrain/VideoSubtitler/commit/e83bba1ecf0f92cf23cd03b7361631982056d27a
[upstream-readme]: https://github.com/GoldYuBrain/VideoSubtitler/blob/e83bba1ecf0f92cf23cd03b7361631982056d27a/README.md
[upstream-main]: https://github.com/GoldYuBrain/VideoSubtitler/blob/e83bba1ecf0f92cf23cd03b7361631982056d27a/main.py
[upstream-transcribe]: https://github.com/GoldYuBrain/VideoSubtitler/blob/e83bba1ecf0f92cf23cd03b7361631982056d27a/subtool/transcribe.py
[upstream-translate]: https://github.com/GoldYuBrain/VideoSubtitler/blob/e83bba1ecf0f92cf23cd03b7361631982056d27a/subtool/translate.py
[upstream-burn]: https://github.com/GoldYuBrain/VideoSubtitler/blob/e83bba1ecf0f92cf23cd03b7361631982056d27a/subtool/burn.py
[upstream-ffmpeg]: https://github.com/GoldYuBrain/VideoSubtitler/blob/e83bba1ecf0f92cf23cd03b7361631982056d27a/subtool/ffmpeg_utils.py
