# 2026-07-22 音訊白噪音降噪原理與調校紀錄

## 結論

本模組以 FFmpeg 的 `afftdn` 濾鏡實作穩定白噪音的頻譜降噪。它先透過
`FFT (Fast Fourier Transform)` 將短時間音訊拆成多個頻率區間，再估算各區間的
噪音能量、降低符合噪音模型的成分，最後轉回時域音訊。

此方法適合持續的嘶聲、電子底噪等穩定寬頻噪音；若噪音與人聲在時間及頻率上
高度重疊，傳統 FFT 濾鏡無法可靠分離，應改用 `speech enhancement` 或
`source separation` 模型。

## 術語解釋 (Terminology)

| 術語 | 本專案中的意義 |
| --- | --- |
| `FFT` | 將時域音訊轉換成頻域資料，供濾鏡逐頻率分析與調整。 |
| `frequency bin` | FFT 產生的單一頻率區間；濾鏡會為每個區間計算不同增益。 |
| `noise profile` | 各頻率區間中預估的噪音分布。白噪音模型假設噪音廣泛分布於頻譜。 |
| `noise floor` | 預估噪音底的音量；數值越接近 `0 dB`，代表假設噪音越大。 |
| `noise reduction` | 被判定為噪音後，允許施加的衰減強度。 |
| `noise tracking` | 隨時間重新估算 noise floor，以處理錄音環境的變化。 |
| `gain smoothing` | 平滑相鄰 frequency bins 的增益，減少零碎金屬聲。 |
| `RMS` | 音訊整體能量的統計值；包含人聲、音樂與噪音，不能直接視為 noise floor。 |
| `artifact` | 降噪造成的副作用，例如金屬聲、機器人聲或水下感。 |

## 處理原理

```mermaid
flowchart LR
    I["輸入音訊"] -->|"切成短時間片段"| W["Window / Frame"]
    W -->|"FFT"| F["各 frequency bin 的能量"]
    F -->|"noise type + noise floor"| N["估算 noise profile"]
    N -->|"計算頻率增益"| G["衰減噪音成分"]
    G -->|"gain smoothing"| S["平滑相鄰頻率"]
    S -->|"Inverse FFT"| O["重建音訊"]
    O -->|"resample + mono"| WAV["16 kHz PCM WAV"]
```

### 1. 頻域轉換

音訊原本是振幅隨時間變化的波形。`afftdn` 對短時間片段執行 FFT，取得各
frequency bin 的能量，因此能分別調整低頻、中頻與高頻，而不是把整體音量一起
降低。

### 2. 噪音估算

`noise_type=white` 使用白噪音頻譜作為先驗模型；`noise_floor` 提供初始噪音
強度。啟用 `track_noise=1` 後，濾鏡會持續調整 noise floor，適合底噪隨錄影位置
或時間改變的素材。

### 3. 頻率增益

濾鏡比較輸入能量與 noise profile，為每個 frequency bin 計算增益。被判定為
噪音的成分最多依 `noise_reduction` 衰減；其餘內容盡量保留。因此降噪不等同於
單純降低整段音量。

### 4. 音訊重建

調整後的頻域資料經 inverse FFT 回到時域，再輸出成轉錄用途的 `16 kHz`、
mono、`pcm_s16le` WAV。

## `afftdn` 主要參數

| 參數 | FFmpeg 範圍與預設 | 調大後的效果 | 主要風險 |
| --- | --- | --- | --- |
| `noise_reduction` (`nr`) | `0.01..97 dB`，預設 `12 dB` | 增加噪音衰減 | 人聲變薄或出現水下感 |
| `noise_floor` (`nf`) | `-80..-20 dB`，預設 `-50 dB` | 將較大聲的成分納入噪音估算 | 可能把安靜語音判為噪音 |
| `noise_type` (`nt`) | 預設 `white` | 改變 noise profile | 錯誤模型會降低效果 |
| `track_noise` (`tn`) | 預設關閉 | 動態追蹤 noise floor | 劇烈變動時可能誤判內容 |
| `gain_smooth` (`gs`) | `0..50`，預設 `0` | 減少隨機 musical-noise artifacts | 過高會模糊音訊細節 |
| `output_mode` (`om`) | `input`、`output`、`noise` | `noise` 可單獨監聽被移除內容 | 若移除內容含清楚人聲，設定過強 |

## `IMG_2906.MOV` 實際紀錄

### 來源

- 路徑：`tmp/IMG_2906.MOV`
- 大小：約 `5.8 GB`
- 時長：`3619.156667` 秒 (`01:00:19`)
- 音軌：`48 kHz`、stereo、AAC

### 第一版：保守模式

```text
noise_reduction=12
noise_floor=-50
noise_type=white
```

結果：主觀聽感的降噪不明顯。原因是 `-50 dB` 假設的 noise floor 太低，只會對
很安靜的底噪施加明顯處理。

### 第二版：強力模式

```text
noise_reduction=35
noise_floor=-20
noise_type=white
track_noise=1
gain_smooth=8
```

輸出：`tmp/IMG_2906-denoised-strong.wav`

- 格式：`16 kHz`、mono、`pcm_s16le`
- 時長：`3619.156000` 秒
- 大小：`115813070` bytes
- 完整解碼檢查：通過

四個抽查區段的整體 mean volume 如下：

| 起點 | 來源 | 強力模式 | 差值 |
| ---: | ---: | ---: | ---: |
| `0s` | `-19.6 dB` | `-22.7 dB` | `-3.1 dB` |
| `600s` | `-18.5 dB` | `-21.1 dB` | `-2.6 dB` |
| `1800s` | `-17.8 dB` | `-19.7 dB` | `-1.9 dB` |
| `3000s` | `-19.5 dB` | `-22.5 dB` | `-3.0 dB` |

這些數字只能證明濾鏡確實改變訊號，不能單獨證明語音品質變好。mean volume 同時
包含人聲與噪音，最終判定仍應比較語音清晰度及 artifacts。

## 調校流程

### 步驟 1 (Step 1)：確認輸入格式與時長

目的：確認存在可處理的音軌，並估算輸出大小與處理時間。

```bash
ffprobe -v error \
  -select_streams a:0 \
  -show_entries format=duration:stream=codec_name,sample_rate,channels \
  -of default=noprint_wrappers=1 \
  tmp/IMG_2906.MOV
```

預期結果：取得 codec、sample rate、channels 與 duration；若沒有 audio stream，
停止處理並回報輸入不含音軌。

### 步驟 2 (Step 2)：抽取代表性預覽

目的：避免每次調整參數都先處理完整影片。至少選取開頭、中段與後段，並包含
說話和沒有說話的區域。

```bash
ffmpeg -ss 600 -t 30 -i tmp/IMG_2906.MOV \
  -vn \
  -af 'afftdn=nr=35:nf=-20:nt=white:tn=1:gs=8' \
  -ar 16000 -ac 1 -c:a pcm_s16le \
  tmp/preview-denoised.wav
```

預期結果：得到約 30 秒的 WAV，可快速比較噪音、人聲清晰度與 artifacts。

### 步驟 3 (Step 3)：依順序調整參數

目的：一次只改變一類因素，才能判斷效果來源。

1. 先調整 `noise_floor`，讓濾鏡能識別實際底噪。
2. 再提高 `noise_reduction`，增加衰減量。
3. 若噪音會隨時間改變，啟用 `track_noise`。
4. 若出現零碎金屬聲，再增加 `gain_smooth`。
5. 監聽 `output_mode=noise`；若被移除的聲音包含清楚人聲，應降低強度。

預期結果：底噪明顯降低，同時對白、人聲尾音與子音仍保持可辨識。

### 步驟 4 (Step 4)：處理完整素材

目的：使用已由預覽確認的參數產生最終 WAV，保留原始影片與較弱版本。

```bash
ffmpeg -hide_banner -y -i tmp/IMG_2906.MOV \
  -vn \
  -af 'afftdn=noise_reduction=35:noise_floor=-20:noise_type=white:track_noise=1:gain_smooth=8' \
  -ar 16000 -ac 1 -c:a pcm_s16le \
  tmp/IMG_2906-denoised-strong.wav
```

預期結果：輸出與來源時長接近的 `16 kHz` mono PCM WAV。

### 步驟 5 (Step 5)：驗證完整性

目的：避免只檢查檔案存在，卻留下截斷或無法解碼的輸出。

```bash
ffprobe -v error \
  -select_streams a:0 \
  -show_entries format=duration,size:stream=codec_name,sample_fmt,sample_rate,channels \
  -of default=noprint_wrappers=1 \
  tmp/IMG_2906-denoised-strong.wav

ffmpeg -v error \
  -i tmp/IMG_2906-denoised-strong.wav \
  -f null -
```

預期結果：`ffprobe` 顯示預期格式及完整時長，完整解碼命令沒有輸出 error。

## 濾鏡選型

| 噪音型態 | 優先方法 | 說明 |
| --- | --- | --- |
| 持續嘶聲、穩定電子底噪 | `afftdn` | FFT 頻譜模型容易調整、速度快。 |
| 隨時間變化的寬頻噪音 | `afftdn` + `track_noise`，或 `afwtdn` adaptive mode | 需要動態更新 noise profile。 |
| 局部波形具有相似上下文的寬頻噪音 | `anlmdn` | 以 Non-Local Means 比較相似 sample contexts。 |
| 固定 `50/60 Hz` hum | notch / equalizer | 針對基頻及 harmonics，不應只靠白噪音模型。 |
| 低頻風聲、碰撞或手持震動 | high-pass + 專用處理 | 先移除無用低頻，再做寬頻降噪。 |
| 噪音與人聲高度重疊 | AI speech enhancement / source separation | 傳統頻譜規則無法可靠判斷相同頻率中的來源。 |

## 目前程式對應與缺口

- `audio.ReduceWhiteNoise`：公開 library API，支援 sample rate、channels、
  `ReductionDB` 與 `NoiseFloorDB`。
- `cmd.DenoiseCmd`：提供 `auth-cli video denoise` 及對應 CLI flags。
- `audio.renderWAV`：集中組合 FFmpeg 參數與輸出 PCM WAV。
- `audio/noise_test.go`：使用合成白噪音驗證處理後 RMS 確實下降。
- 目前 API/CLI 尚未公開 `track_noise` 與 `gain_smooth`；本次強力模式透過直接
  FFmpeg 命令執行。若強力模式成為正式產品行為，應將兩個參數加入
  `WhiteNoiseOptions` 與 CLI，並補參數邊界測試。

## 參考資料

- [FFmpeg `afftdn` filter](https://ffmpeg.org/ffmpeg-filters.html#afftdn)
- [FFmpeg `afwtdn` filter](https://ffmpeg.org/ffmpeg-filters.html#afwtdn)
- [FFmpeg `anlmdn` filter](https://ffmpeg.org/ffmpeg-filters.html#anlmdn)
