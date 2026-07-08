package mailer

import (
	"fmt"
	"html"
	"strings"
)

// Email is the content poured into the branded layout.
type Email struct {
	AppName    string // header bar text, e.g. "POS System"
	Title      string // headline inside the card
	Intro      string // paragraph under the headline (plain text)
	ButtonText string // optional call-to-action
	ButtonURL  string
	BodyHTML   string // optional extra block (already-safe HTML, e.g. a <ul>)
	FooterNote string // e.g. "If you didn't request this, you can ignore it."
}

// Render wraps content in a clean, email-client-safe layout (tables +
// inline styles) so transactional mail reads as legitimate: brand
// header, card, prominent button, copyable fallback link, footer.
func Render(e Email) string {
	var button, fallback string
	if e.ButtonURL != "" && e.ButtonText != "" {
		safeURL := html.EscapeString(e.ButtonURL)
		button = fmt.Sprintf(`
			<tr><td align="center" style="padding:28px 40px 8px;">
				<a href="%s" style="background-color:#16a34a;border-radius:8px;color:#ffffff;display:inline-block;font-size:16px;font-weight:bold;line-height:48px;text-align:center;text-decoration:none;min-width:220px;padding:0 28px;">%s</a>
			</td></tr>`, safeURL, html.EscapeString(e.ButtonText))
		fallback = fmt.Sprintf(`
			<tr><td style="padding:8px 40px 0;color:#6b7280;font-size:12px;line-height:18px;">
				If the button doesn't work, copy this link into your browser:<br>
				<a href="%s" style="color:#16a34a;word-break:break-all;">%s</a>
			</td></tr>`, safeURL, safeURL)
	}

	extra := ""
	if e.BodyHTML != "" {
		extra = fmt.Sprintf(`<tr><td style="padding:16px 40px 0;color:#374151;font-size:15px;line-height:24px;">%s</td></tr>`, e.BodyHTML)
	}

	footer := e.FooterNote
	if footer == "" {
		footer = "You received this email because of activity on your account."
	}

	return strings.TrimSpace(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f3f4f6;font-family:Arial,Helvetica,sans-serif;">
	<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f3f4f6;padding:32px 12px;">
		<tr><td align="center">
			<table role="presentation" width="560" cellpadding="0" cellspacing="0" style="max-width:560px;width:100%%;">
				<tr><td style="background-color:#16a34a;border-radius:12px 12px 0 0;padding:20px 40px;">
					<span style="color:#ffffff;font-size:18px;font-weight:bold;letter-spacing:0.3px;">%s</span>
				</td></tr>
				<tr><td style="background-color:#ffffff;border-radius:0 0 12px 12px;">
					<table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
						<tr><td style="padding:32px 40px 0;color:#111827;font-size:20px;font-weight:bold;line-height:28px;">%s</td></tr>
						<tr><td style="padding:12px 40px 0;color:#374151;font-size:15px;line-height:24px;">%s</td></tr>
						%s%s%s
						<tr><td style="padding:28px 40px 32px;color:#9ca3af;font-size:12px;line-height:18px;border-top:1px solid #f3f4f6;">%s</td></tr>
					</table>
				</td></tr>
				<tr><td align="center" style="padding:16px 8px;color:#9ca3af;font-size:12px;">© %s — this is an automated message, replies are not monitored.</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>`,
		html.EscapeString(e.AppName),
		html.EscapeString(e.Title),
		html.EscapeString(e.Intro),
		extra, button, fallback,
		html.EscapeString(footer),
		html.EscapeString(e.AppName),
	))
}
