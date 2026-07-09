# Marketing assets

Promo material for the **POS System** Facebook page (facebook.com/webdevbot).

> ⚠️ These are ready-to-use files, but they are **not auto-posted**. Posting to
> Facebook and recording the video are manual steps (done by you) — see below.

## `social/week-1/` — a week of Facebook posts

Seven square **1080×1080** post images, one per day, plus copy:

| Day | Image | Theme |
|-----|-------|-------|
| Mon | `post-1.png` | Launch / brand intro |
| Tue | `post-2.png` | Point of Sale — ring up a sale |
| Wed | `post-3.png` | Payments & receipts (GCash/Maya) |
| Thu | `post-4.png` | Kitchen Display |
| Fri | `post-5.png` | Inventory & recipes |
| Sat | `post-6.png` | Sales analytics |
| Sun | `post-7.png` | Pricing & call-to-action |

- **`captions.md`** — the caption + hashtags for each day, plus a posting
  schedule and step-by-step "how to post".
- The `.html` files are the editable sources; the `.png` files are what you
  upload. To tweak text/colors, edit the HTML and re-export (open the file in a
  browser at a 1080×1080 window and screenshot, or ask me to re-render).

**To post:** Facebook → your Page → *Create post* → upload the day's PNG →
paste that day's caption from `captions.md` → Publish.

## `video/presentation.html` — animated feature walkthrough

A self-contained, auto-playing "video" that walks through the whole product:
intro → what it does → getting started (3 steps) → POS → payments → kitchen
display → inventory → staff & customers → analytics → multi-business → pricing →
call-to-action. Warm branded design, 16:9.

**Controls:** `Space` pause/resume · `←` `→` navigate · `F` fullscreen · click to advance.

**To turn it into a shareable video (.mp4):**
1. Open `video/presentation.html` in Chrome and press **F** for fullscreen.
2. Screen-record it (Windows: **Win + G** → Game Bar → Record; or OBS Studio for
   higher quality). It auto-advances through all 12 slides in ~75 seconds.
3. Stop the recording when it loops back to the intro — you now have an .mp4 to
   upload to Facebook, Reels, or TikTok.
   - For a voiceover, record narration over it (a suggested script per slide can
     be added on request).

## Notes
- All prices shown (₱800 / ₱8,000) match the live app defaults. If you change
  them in the super-admin billing settings, update the images/slides to match.
- Fonts (Fraunces + Inter) load from Google Fonts, so keep an internet
  connection when exporting images or recording the video.
