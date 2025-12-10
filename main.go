package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// Scanner always appends a newline (<CR> / Enter).
// All Code values are complete messages and do NOT include newline characters.

type ChatMsg struct {
	Code        string // exact text encoded in the QR code (no newline)
	Label       string // short label under QR code
	Description string // longer explanation under the label
}

// 36 messages => 4 x 9 grid.
var Messages = []ChatMsg{
	// --- Status / presence ---
	{"On my way, be there soon.", "On my way", "Quick status: in transit, joining soon."},
	{"BRB – back in 5 minutes.", "BRB 5", "Short break, back in 5."},
	{"AFK for a bit, I’ll respond when I’m back.", "AFK", "Away-from-keyboard notice."},
	{"Stepping out, please continue without me.", "Stepping out", "Let others know they can continue."},

	// --- General acknowledgements ---
	{"Got it, thanks!", "Got it", "Simple acknowledgement."},
	{"Thanks for the heads up.", "Heads up", "Acknowledges a warning or FYI."},
	{"Thanks, I’ll take a look.", "I'll look", "You’re taking ownership to investigate."},
	{"Thanks, this is really helpful.", "Helpful", "Extra appreciative acknowledgement."},

	// --- Requesting info ---
	{"Can you please share a screenshot of the issue?", "Screenshot?", "Ask for a screenshot."},
	{"Can you please paste the error message here?", "Error msg?", "Ask for the exact error message."},
	{"Which OS / browser / version are you using?", "Env details?", "Ask for environment details."},
	{"Can you describe the steps to reproduce this?", "Repro steps?", "Ask for a clear repro."},

	// --- Triage / queueing ---
	{"I’ve noted this down – it might take a little while before I can dig in.", "Noted, queued", "You’ve captured the issue, not immediate."},
	{"I’m looking into this now.", "Looking now", "You’re actively investigating."},
	{"This looks important – I’m prioritising it.", "Prioritising", "You’re giving it priority."},
	{"Thanks – I think this is a duplicate of an existing issue, I’ll cross-link it.", "Duplicate", "Triage as duplicate."},

	// --- Moderation / boundaries ---
	{"Let’s keep the conversation respectful and on-topic, please.", "Respectful", "Gentle moderation reminder."},
	{"This thread is getting heated – please take a break and come back later.", "Cool down", "Ask people to cool off."},
	{"Please move this conversation to the appropriate channel.", "Wrong channel", "Redirect to the right channel."},
	{"I’m going to lock this thread if the tone doesn’t improve.", "Tone warning", "Clear warning for behaviour."},

	// --- Dev / infra / deploy chatter ---
	{"Deploying to production now – expect a brief disruption.", "Deploying now", "Deploy in progress notice."},
	{"Deployment finished successfully.", "Deploy OK", "Deployment success message."},
	{"We’re rolling back this deployment due to issues.", "Rolling back", "Rollback notice."},
	{"We’re investigating an issue in production – updates soon.", "Prod issue", "Production incident notice."},

	// --- Support / closing loops ---
	{"I believe this should be fixed now – can you confirm?", "Please confirm", "Ask user to verify fix."},
	{"Closing this out for now – feel free to reopen if it happens again.", "Closing", "Gentle closure message."},
	{"Thanks for your patience while we sorted this out.", "Thanks for patience", "Thank users after delays."},
	{"Thanks again for the report – this really helps us improve.", "Thanks for report", "Reinforce helpfulness."},

	// --- Generic “nice” utilities ---
	{"Good morning! 👋", "GM", "Quick morning greeting."},
	{"Good night, talk to you all tomorrow.", "GN", "Quick goodnight."},
	{"Congratulations, that’s awesome news! 🎉", "Congrats", "Celebrate good news."},
	{"Happy birthday! 🎂", "Birthday", "Birthday wish."},

	// --- Meta / fallback messages ---
	{"I don’t have enough context yet – can you give me a bit more detail?", "More context?", "Ask for more info, generic."},
	{"I might be slow to respond for a while, but I am reading everything.", "Slow replies", "Set expectation for slower replies."},
	{"I’ve created an internal note/ticket for this, and we’ll track it from there.", "Internal ticket", "Let them know it’s being tracked."},
	{"If anyone else experiences this, please react to this message so we can gauge impact.", "React to gauge", "Ask for reactions to measure impact."},
}

// font cache so we only parse Go Regular once per size.
var fontCache = map[float64]font.Face{}

func main() {
	// A4 @ 300 DPI
	const dpi = 300
	const a4WidthInches = 8.27
	const a4HeightInches = 11.69

	width := int(a4WidthInches * dpi)
	height := int(a4HeightInches * dpi)

	dc := gg.NewContext(width, height)

	// Background
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	margin := 80.0

	// Title
	dc.SetColor(color.Black)
	dc.SetFontFace(mustGoRegularFace(24))
	title := "Chat QR Codes – One Scan = One Message"
	dc.DrawStringAnchored(title, float64(width)/2, margin/2, 0.5, 0.5)

	// Layout: 4 columns, N rows
	cols := 4
	rows := int(math.Ceil(float64(len(Messages)) / float64(cols)))

	top := margin
	bottom := float64(height) - margin
	left := margin
	right := float64(width) - margin

	cellWidth := (right - left) / float64(cols)
	cellHeight := (bottom - top) / float64(rows)

	// QR codes are square; size them to fit comfortably in each cell.
	qrSize := int(math.Min(cellWidth, cellHeight) * 0.6)

	for i, msg := range Messages {
		col := i % cols
		row := i / cols

		x := left + float64(col)*cellWidth
		y := top + float64(row)*cellHeight

		cx := x + cellWidth/2

		// Light cell boundary
		dc.SetLineWidth(0.4)
		dc.SetColor(color.RGBA{R: 230, G: 230, B: 230, A: 255})
		dc.DrawRectangle(x, y, cellWidth, cellHeight)
		dc.Stroke()

		// --- QR generation ---
		raw, err := qr.Encode(msg.Code, qr.M, qr.Auto)
		if err != nil {
			log.Printf("QR encode error for %q: %v", msg.Code, err)
			continue
		}

		scaled, err := barcode.Scale(raw, qrSize, qrSize)
		if err != nil {
			log.Printf("QR scale error for %q: %v", msg.Code, err)
			continue
		}

		// Draw QR near the top of the cell
		bx := cx - float64(scaled.Bounds().Dx())/2
		by := y + 6
		dc.DrawImage(scaled, int(bx), int(by))

		// Label under QR
		labelY := by + float64(qrSize) + 8
		dc.SetColor(color.Black)
		dc.SetFontFace(mustGoRegularFace(11))
		label := msg.Label
		if label == "" {
			label = msg.Code
		}
		dc.DrawStringAnchored(label, cx, labelY, 0.5, 0)

		// Description under label
		descY := labelY + 12
		dc.SetFontFace(mustGoRegularFace(8))
		dc.DrawStringWrapped(msg.Description, x+6, descY, 0, 0, cellWidth-12, 1.3, gg.AlignCenter)
	}

	// --- Footer: repo QR + text ---
	footerText := "https://github.com/arran4/chat-barcodes"

	footerRaw, err := qr.Encode(footerText, qr.M, qr.Auto)
	if err != nil {
		log.Printf("QR encode error for footer: %v", err)
	} else {
		// Keep the QR comfortably inside the bottom margin
		footerSize := int(math.Min(float64(width)*0.18, margin*0.8))

		footerScaled, err := barcode.Scale(footerRaw, footerSize, footerSize)
		if err != nil {
			log.Printf("QR scale error for footer: %v", err)
		} else {
			// Place QR above bottom margin, centered horizontally
			fbX := float64(width)/2 - float64(footerScaled.Bounds().Dx())/2
			fbY := float64(height) - margin - float64(footerSize) - 10
			dc.DrawImage(footerScaled, int(fbX), int(fbY))

			// Footer text just above the very bottom of the page
			textY := float64(height) - 12
			dc.SetColor(color.Black)
			dc.SetFontFace(mustGoRegularFace(9))
			dc.DrawStringAnchored(footerText, float64(width)/2, textY, 0.5, 0)
		}
	}

	out := "chat-qr-a4.png"
	if err := dc.SavePNG(out); err != nil {
		log.Fatalf("failed to save PNG: %v", err)
	}

	fmt.Println("Saved:", out)
}

// mustGoRegularFace returns a Go Regular font.Face at the given size,
// always using the embedded goregular TTF.
func mustGoRegularFace(size float64) font.Face {
	if face, ok := fontCache[size]; ok {
		return face
	}

	fnt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatalf("failed to parse goregular TTF: %v", err)
	}

	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("failed to create goregular face (size=%.1f): %v", size, err)
	}

	fontCache[size] = face
	return face
}
